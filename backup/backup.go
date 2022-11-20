package backup

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5/config"
	"os"
	"path"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/d5/tengo/v2"
	"github.com/go-git/go-git/v5"
	"github.com/gookit/color"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()
var log = logrus.NewEntry(logger)

func SetLogger(nLogger *logrus.Logger) {
	logger = nLogger
}

func SetLog(entty *logrus.Entry) {
	log = entty
}

type Provider int

const (
	GitHub Provider = iota
	GitLab
	//TODO: expand if you have more implementations ;)
)

type Orphaned int

const (
	IgnoreOrphaned = iota
	PullOrphaned
	RemoveOrphaned
)

type Account struct {
	Name       string   `yaml:"name"`
	Provider   Provider `yaml:"provider"`
	Token      string   `yaml:"token"`
	Args       []string `yaml:"args"`
	BlackList  []string `yaml:"blacklist"`
	FilterList []string `yaml:"filters"`
}

type Config struct {
	Repository string    `yaml:"repository"`
	Accounts   []Account `yaml:"accounts"`

	OverwriteOnConflict bool     `yaml:"overwrite_on_conflict"`
	HandleOrphaned      Orphaned `yaml:"handle_orphaned"`
}

type GoGitBackup struct {
	clients  []client
	config   *Config
	repos    []Repository
	errorLog *os.File
}

type Visibility int

const (
	Public Visibility = iota
	Private
	Internal
)

type Repository struct {
	CloneUrl     string
	Name         string
	Size         int64
	CreatedAt    time.Time
	Owner        bool
	Member       bool
	Visibility   Visibility
	ProviderName string
	Archived     bool
}

type client interface {
	Init() error
	List() ([]Repository, error)
	Name() string
	RegisterFilter(filters []*tengo.Script)
}

func (c *GoGitBackup) _info(bar *pb.ProgressBar, msg string) {
	bar.Set("info", fmt.Sprintf("%50.50s", msg)).Set("warn", "")
}

func (c *GoGitBackup) _error(bar *pb.ProgressBar, msg string) {
	bar.Set("warn", fmt.Sprintf("%50.50s", msg)).Set("info", "")
	log.Debugf(msg)
	if c.errorLog != nil {
		_, _ = c.errorLog.WriteString(msg + "\n")
	}
}

func NewGoBackup(cnf *Config, logFile *os.File) (*GoGitBackup, error) {
	repositoryLocation, err := os.Stat(cnf.Repository)
	if err != nil {
		log.Debugf("Failed to obtain fileInfo for %s, %+v", cnf.Repository, err)
		return nil, err
	}

	if !repositoryLocation.IsDir() {
		log.Debugf("%s is not a directory", cnf.Repository)
		return nil, fmt.Errorf("%s is not a directory", cnf.Repository)
	}

	clients := make([]client, 0)

	for _, account := range cnf.Accounts {
		filters := make([]*tengo.Script, 0)
		for _, filterCode := range account.FilterList {
			filters = append(filters, tengo.NewScript([]byte(filterCode)))
		}

		switch account.Provider {

		case GitHub:
			client := &_githubClient{
				ctx:   context.Background(),
				Token: account.Token,
				name:  account.Name,
			}
			if account.Args != nil && len(account.Args) > 0 {
				client.User = account.Args[0]
			}
			client.RegisterFilter(filters)
			clients = append(clients, client)

		case GitLab:
			client := &_gitlabClient{
				Token: account.Token,
				name:  account.Name,
			}
			if account.Args != nil && len(account.Args) > 0 {
				client.BaseURL = account.Args[0]
			}
			client.RegisterFilter(filters)
			clients = append(clients, client)
			//TODO: extend here if you add a new provider
		}
	}

	return &GoGitBackup{
		config:   cnf,
		clients:  clients,
		errorLog: logFile,
	}, nil
}

func (c *GoGitBackup) Do() {
	if c.Check() != nil {
		return
	}

	tmpl := `{{ bar . "<" "-" (cycle . "↖" "↗" "↘" "↙" ) "." ">"}} {{speed . | white }} {{percent .}} {{string . "info" | green}}  {{string . "warn" | red}}`

	bar := pb.ProgressBarTemplate(tmpl).New(len(c.repos)).SetWriter(os.Stdout).Start()

	updated := make(map[string]struct{}, 0)
	for _, repo := range c.repos {
		bar.Increment()

		targetLocation := path.Join(c.config.Repository, repo.Name)
		updated[targetLocation] = struct{}{}
		if _, err := os.Stat(targetLocation); err != nil {
			//we assume that the file does not exist and proceed with pulling
			c._info(bar, fmt.Sprintf("Cloning %s into %s", repo.Name, targetLocation))
			_, err := git.PlainClone(targetLocation, false, &git.CloneOptions{
				URL: repo.CloneUrl,
			})

			if err != nil {
				c._error(bar, fmt.Sprintf("Failed to clone repo for %s - %+v", repo.Name, err))
			}
		} else {
			c._info(bar, fmt.Sprintf("Pulling %s", targetLocation))
			err := c.pull(repo)
			if err != nil {
				c._error(bar, fmt.Sprintf("Failed to clone pull for %s - %+v", repo.Name, err))
			}
		}

	}
	bar.Finish()

	if c.config.HandleOrphaned != IgnoreOrphaned {
		orphaned := c.findOrphaned(updated)
		if len(orphaned) > 0 {
			bar = pb.ProgressBarTemplate(tmpl).New(len(c.repos)).SetWriter(os.Stdout).Start()
			for _, orphan := range orphaned {
				if c.config.HandleOrphaned == RemoveOrphaned {
					err := os.RemoveAll(orphan)
					c._info(bar, fmt.Sprintf("Removed orphaned repo %s - %v", orphan, err))
				} else if c.config.HandleOrphaned == PullOrphaned {
					err := _pull(orphan)
					if err != nil && err != git.NoErrAlreadyUpToDate {
						c._error(bar, fmt.Sprintf("Failed to pull orphaned repo %s - %v", orphan, err))
					}
					c._info(bar, fmt.Sprintf("Pulled orphaned repo %s", orphan))
				}
			}
			bar.Finish()
		}
	}
}

func (c *GoGitBackup) findOrphaned(known map[string]struct{}) []string {
	return find(c.config.Repository, known)
}

// find all git directories that are not in the known map recursively starting from the given root path
func find(root string, known map[string]struct{}) []string {
	orphaned := make([]string, 0)
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.IsDir() {
			edir := path.Join(root, e.Name())
			if _, err := os.Stat(path.Join(edir, ".git")); err != nil {
				orphaned = append(orphaned, find(edir, known)...)
			} else {
				if _, ok := known[edir]; !ok {
					orphaned = append(orphaned, edir)
				}
			}
		}
	}
	return orphaned
}

func (c *GoGitBackup) pull(repo Repository) error {
	targetLocation := path.Join(c.config.Repository, repo.Name)

	err := _pull(targetLocation)

	if err == git.NoErrAlreadyUpToDate {
		return nil
	} else if err != nil {
		if c.config.OverwriteOnConflict {
			log.Infof("Deleting %s due to conflict", targetLocation)

			err = os.Rename(targetLocation, targetLocation+"_conflict")
			if err != nil {
				return fmt.Errorf("failed to delete %s due to conflict", targetLocation)
			}

			_, err := git.PlainClone(targetLocation, false, &git.CloneOptions{URL: repo.CloneUrl})
			if err != nil {
				log.Errorf("failed to clone repo %s, reverting. %+v", repo.Name, err)
				err = os.Rename(targetLocation+"_conflict", targetLocation)
				if err != nil {
					return fmt.Errorf("failed to revert check %s_conflict for original", targetLocation)
				}
			}
			log.Infof("Overwritten %s", repo.Name)
		}
		return fmt.Errorf("failed to fetch repo:%+v", err)
	}
	return nil
}

func _pull(targetLocation string) error {
	r, err := git.PlainOpen(targetLocation)
	if err != nil {
		return fmt.Errorf("failed to open repo: %+v", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to enter repo: %+v", err)
	}

	err = w.Pull(&git.PullOptions{
		Force: true,
	})
	return err
}

func (c *GoGitBackup) Close() {
	if c.errorLog != nil {
		_ = c.errorLog.Close()
	}
}

func (c *GoGitBackup) Check() error {
	repos := make([]Repository, 0)

	for _, client := range c.clients {
		err := client.Init()
		if err != nil {
			color.Style{color.FgBlack, color.BgGray}.Printf("Failed to init client %s\n", client.Name())
			color.Style{color.FgBlack, color.BgGray}.Printf("Reason:%+v", err)
			return err
		}

		repo, err := client.List()
		if err != nil {
			color.Style{color.FgBlack, color.BgGray}.Printf("Failed to list repo for %s\n", client.Name())
			color.Style{color.FgBlack, color.BgGray}.Printf("Reason:%+v", err)
			return err
		}

		repos = append(repos, repo...)
	}

	c.repos = repos

	const TableFormat = "| %10.10s\t| %60.60s\t| %10.10s\t| %10.10s\t|\n"

	fmt.Printf("Found the following repositories:\n")
	fmt.Printf(TableFormat, "Provider", "Name", "CreatedAt", "Size")

	for _, repo := range c.repos {
		fmt.Printf(TableFormat, repo.ProviderName, repo.Name, repo.CreatedAt, fmt.Sprintf("%10.0d", repo.Size))
	}

	return nil
}

func (c *GoGitBackup) Update() error {
	err := c.Check()
	if err != nil {
		return fmt.Errorf("failed to check repo: %+v", err)
	}
	for _, repo := range c.repos {
		targetLocation := path.Join(c.config.Repository, repo.Name)

		if _, err := os.Stat(targetLocation); err != nil {
			continue
		} else {
			r, err := git.PlainOpen(targetLocation)
			if err != nil {
				return fmt.Errorf("failed to open repo: %+v", err)
			}

			cnf, err := r.Config()
			if err != nil {
				return fmt.Errorf("failed to open repo: %+v", err)
			}

			rmf := cnf.Remotes["origin"]

			if rmf.URLs[0] != repo.CloneUrl {
				log.Infof("%s outdated updateding 'origin'", repo.Name)
				rmf.Name = "old-remote"
				cnf.Remotes["old-remote"] = rmf
				cnf.Remotes["origin"] = &config.RemoteConfig{
					Name: "origin",
					URLs: []string{repo.CloneUrl},
				}
				err = r.SetConfig(cnf)
				if err != nil {
					return fmt.Errorf("failed to set config: %+v", err)
				}
			} else {
				continue
			}

		}

	}

	return nil
}

func filter(repo Repository, filters []*tengo.Script) bool {
	for i, filter := range filters {
		if !apply(filter, repo) {
			log.Debugf("%s was filtered due to filter[%d]", repo.Name, i)
			return false
		}
	}
	return true
}

func apply(filter *tengo.Script, repo Repository) bool {

	_ = filter.Add("owner", repo.Owner)
	_ = filter.Add("member", repo.Member)
	_ = filter.Add("visibility", int(repo.Visibility))
	_ = filter.Add("size", repo.Size)
	_ = filter.Add("name", repo.Name)

	run, err := filter.Run()
	if err != nil {
		color.Style{color.FgRed, color.BgDarkGray}.Printf("Failed apply filte rule for %s cause:%+v\n", repo.Name, err)
	}

	return run.Get("r").Bool()
}
