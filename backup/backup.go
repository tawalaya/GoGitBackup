package backup

import (
	"context"
	"fmt"

	"os"
	"path"
	"time"

	"gopkg.in/src-d/go-git.v4"

	"github.com/cheggaaa/pb/v3"
	"github.com/gookit/color"

	"github.com/d5/tengo/v2"
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

type Account struct {
	Name      string   `yaml:"name"`
	Provider  Provider `yaml:"provider"`
	Token     string   `yaml:"token"`
	Args      []string `yaml:"args"`
	BlackList []string `yaml:"blacklist"`
	FilterList []string  `yaml:"filters"`
}

type Config struct {
	Repository string    `yaml:"repository"`
	Accounts   []Account `yaml:"accounts"`
}

type GoGitBackup struct {
	clients []client
	config  *Config
	repos   []Repository
}

type Visibility int

const (
	Public Visibility = iota
	Private
	Internal
)

type Repository struct {
	CloneUrl   string
	Name       string
	Size       int64
	CreatedAt  time.Time
	Owner      bool
	Member     bool
	Visibility Visibility
}

type client interface {
	Init() error
	List() ([]Repository, error)
	Name() string
	RegisterFilter(filters []*tengo.Script)
}

func _info(bar *pb.ProgressBar, msg string) {
	bar.Set("info", fmt.Sprintf("%30.30s", msg)).Set("warn", "")
}

func _error(bar *pb.ProgressBar, msg string) {
	bar.Set("warn", fmt.Sprintf("%30.30s", msg)).Set("info", "")
}

func NewGoBackup(cnf *Config) (*GoGitBackup, error) {
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
		config:  cnf,
		clients: clients,
	}, nil
}

func (c *GoGitBackup) Do() {
	if c.Check() != nil {
		return
	}

	tmpl := `{{ bar . "<" "-" (cycle . "↖" "↗" "↘" "↙" ) "." ">"}} {{speed . | white }} {{percent .}} {{string . "info" | green}}  {{string . "warn" | red}}`

	bar := pb.ProgressBarTemplate(tmpl).New(len(c.repos)).SetWriter(os.Stdout).Start()

	for _, repo := range c.repos {
		bar.Increment()

		targetLocation := path.Join(c.config.Repository, repo.Name)

		if _, err := os.Stat(targetLocation); err != nil {
			//we assume that the file dose not exist and proceed with pulling
			_info(bar, fmt.Sprintf("Cloning %s into %s", repo.Name, targetLocation))
			_, err := git.PlainClone(targetLocation, false, &git.CloneOptions{
				URL: repo.CloneUrl,
			})

			if err != nil {
				_error(bar, fmt.Sprintf("Failed to clone repo for %s", repo.Name))
				log.Debugf("Failed to clone repo for %s Reason:%+v\n", repo.Name, err)
			}
		} else {
			_info(bar, fmt.Sprintf("Pulling %s", targetLocation))

			r, err := git.PlainOpen(targetLocation)
			if err != nil {
				_error(bar, fmt.Sprintf("Failed to open repo for %s", repo.Name))
				log.Debugf("Failed to clone repo for %s Reason:%+v\n", repo.Name, err)
				//if we do it strict we should fail here!
				continue
			}
			err = r.Fetch(&git.FetchOptions{})
			if err != nil {
				_error(bar, fmt.Sprintf("Failed to fetch repo for %s Reason:%+v\n", repo.Name, err))
				log.Debugf("Failed to fetch repo for %s Reason:%+v\n", repo.Name, err)
			}
		}

	}

	bar.Finish()
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

	fmt.Printf("Found the following repositories:\n")
	fmt.Printf("| %60.10s\t| %10.10s\t| %10.10s\t|\n", "Name", "CreatedAt", "Size")

	for _, repo := range c.repos {
		fmt.Printf("| %60.10s\t| %10.10s\t| %10.0d\t|\n", repo.Name, repo.CreatedAt, repo.Size)
	}

	return nil
}

func filter(repo Repository,filters []*tengo.Script) bool {
	for _, filter := range filters {
		if !apply(filter, repo) {
			return false
		}
	}
	return true
}

func apply(filter *tengo.Script, repo Repository) bool {

	_ = filter.Add("owner", repo.Owner)
	_ = filter.Add("member", repo.Member)
	_ = filter.Add("visibility", repo.Visibility)
	_ = filter.Add("size", repo.Size)
	_ = filter.Add("name", repo.Name)

	run, err := filter.Run()
	if err != nil {
		color.Style{color.FgRed, color.BgDarkGray}.Printf("Failed apply filte rule for %s cause:%+v\n", repo.Name, err)
	}

	return run.Get("r").Bool()
}
