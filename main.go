/*
 * Copyright 2020 Sebastian Werner
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"golang.org/x/oauth2"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/xanzy/go-gitlab"
)

var (
	Build string
)

var logger = logrus.New()
var log *logrus.Entry

func init() {
	if Build == "" {
		Build = "Debug"
	}
	logger.Formatter = new(prefixed.TextFormatter)
	logger.SetLevel(logrus.DebugLevel)
	log = logger.WithFields(logrus.Fields{
		"prefix": "git-backup",
		"build":  Build,
	})
}


type Provider int

const (
	GitHub Provider = iota
	GitLab
	//TODO: expand if you have more implementations ;)
)

type Account struct {
	Name string  `yaml:"name"`
	Provider Provider  `yaml:"provider"`
	Token string  `yaml:"token"`
	Args []string  `yaml:"args"`
}

type Config struct {
	Repository string `yaml:"repository"`
	Accounts []Account `yaml:"accounts"`
}

//TODO split into separate files and clean up
func main() {
	app := &cli.App{
		Name: "gitback",
		Usage: "Utility to backup your git(Hub|lab) accounts.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Load configuration from `FILE`",
				Value:   "./config.yml",
			},
			&cli.BoolFlag{
				Name:"verbose",
				Aliases:[]string{"v"},
				Usage:"Enables verbose logging",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "backup",
				Aliases: []string{"b"},
				Usage:   "performs backup of all git(hub/lab) accounts that can be accessed.",
				Action:  func(c *cli.Context) error {
					client := preflight(c)
					client.Do()
					return nil
				},
			},
			{
				Name:    "check",
				Aliases: []string{"c"},
				Usage:   "check what we can backup using this utility and also validates your config ;)",
				Action:  func(c *cli.Context) error {
					client := preflight(c)
					client.Check()
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}


func preflight(c *cli.Context) *GoGitBackup {

	bytes,err := ioutil.ReadFile(c.String("config"))

	if err != nil{
		log.Fatalf("failed to read config at %s %+v",c.String("config"),err)
	}

	var config Config

	err = yaml.Unmarshal(bytes, &config)

	if err != nil{
		log.Fatalf("failed to parse config, %+v",err)
	}

	if c.Bool("verbose") {
		logger.SetLevel(logrus.DebugLevel)
		log.Debugf("using config:\n %+v",config)
	}
	client,err := NewGoBackup(&config)

	if err != nil{
		log.Fatalf("failed to create backup client due to %+v",err)
	}

	return client

}

type GoGitBackup struct {
	clients []client
	config  *Config
	repos   []Repository
}

type Repository struct {
	CloneUrl string
	Name string
	Size int64
	CreatedAt time.Time
}

type client interface {
	Init() error
	List() ([]Repository,error)
	Name() string
}

type _githubClient struct {
	ctx context.Context
	client *github.Client
	Token string
	User string
	name string

}

func (c *_githubClient) Name() string {
	return c.name
}


func (c *_githubClient) Init() error {

	ctx := context.Background()
	c.ctx = ctx

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	c.client = client

	return nil
}


func (c *_githubClient) List() ([]Repository,error) {
	
	list,res,err := c.client.Repositories.List(c.ctx,"",&github.RepositoryListOptions{
		Visibility:  "all",
		//Affiliation: "owner",
		//Type:        "all",
		Sort:        "created",
	})

	if err != nil{
		log.Debug("failed to list GitHub repositories reason %+v",res)
		return nil,err
	}

	repoList := make([]Repository,0)

	for _, repo := range list {
		log.Infof("got %s size %d",repo.GetFullName(),repo.GetSize())

		url := repo.GetCloneURL()

		if c.User != ""{
			url = strings.Replace(url,"https://",fmt.Sprintf("https://%s:%s@",c.User,c.Token),-1)
		}

		repoList = append(repoList,
			Repository{
				CloneUrl: url,
				Name:      repo.GetFullName(),
				Size:      int64(repo.GetSize()),
				CreatedAt: repo.GetCreatedAt().UTC(),
			})

	}

	return repoList,nil
}


type _gitlabClient struct {
	Token string
	BaseURL string
	client *gitlab.Client
	name string
}

func (c *_gitlabClient) Init() error{

	git := gitlab.NewClient(nil,c.Token)
	if c.BaseURL != "" {
		err := git.SetBaseURL(c.BaseURL)
		if err != nil{
			log.Debug("failed to set base url to %s, %+v",c.BaseURL,err)
			return err
		}

	}
	c.client = git

	return nil
}


func (c *_gitlabClient) List() ([]Repository,error) {
	projects,res,err := c.client.Projects.ListProjects(&gitlab.ListProjectsOptions{})

	if err != nil{
		log.Debug("failed to list GitHub repositories reason %+v",res)
		return nil,err
	}

	repoList := make([]Repository,0)
	for _,project := range projects{
		var size int64
		if project.Statistics != nil{
			size = project.Statistics.StorageSize
		} else {
			size = -1
		}

		repoList = append(repoList,Repository{
			CloneUrl:  strings.Replace(project.HTTPURLToRepo,"https://",fmt.Sprintf("https://oauth2:%s@",c.Token),-1),
			Name:      strings.ReplaceAll(strings.ReplaceAll(project.NameWithNamespace," / ","/")," ","_"),
			Size:      size,
			CreatedAt: *project.CreatedAt,
		})
	}

	return repoList,nil
}

func (c *_gitlabClient) Name() string {
	return c.name
}



func NewGoBackup(cnf *Config) (*GoGitBackup,error){
	repositoryLocation,err := os.Stat(cnf.Repository)
	if err != nil{
		log.Debugf("Failed to obtain fileInfo for %s, %+v", cnf.Repository,err)
		return nil,err
	}

	if !repositoryLocation.IsDir() {
		log.Debugf("%s is not a directory", cnf.Repository)
		return nil,fmt.Errorf("%s is not a directory", cnf.Repository)
	}

	clients := make([]client,0)

	for _, account := range cnf.Accounts {
		switch account.Provider {

		case GitHub:
			client := &_githubClient{
				ctx:    context.Background(),
				Token:  account.Token,
				name:account.Name,
			}
			if account.Args != nil && len(account.Args) > 0 {
				client.User = account.Args[0]
			}
			clients = append(clients,client)

		case GitLab:
			client := &_gitlabClient{
				Token:  account.Token,
				name:account.Name,
			}
			if account.Args != nil && len(account.Args) > 0 {
				client.BaseURL = account.Args[0]
			}
			clients = append(clients,client)
		//TODO: extend here if you add a new provider
		}
	}


	return &GoGitBackup{
		config:  cnf,
		clients: clients,


	},nil
}

func (c *GoGitBackup) Do(){
	if c.Check() != nil{
		return
	}

	for _, repo := range c.repos {

		targetLocation := path.Join(c.config.Repository,repo.Name)



		if _,err := os.Stat(targetLocation);err != nil{
			//we assume that the file dose not exist and proceed with pulling
			color.Printf("Cloning %s into %s\n",repo.Name,targetLocation)
			_,err := git.PlainClone(targetLocation,false,&git.CloneOptions{
				URL:           repo.CloneUrl,
			})

			if err != nil{
				color.Style{color.FgRed}.Printf("Failed to clone repo for %s Reason:%+v\n",repo.Name,err)
			}
		} else {
			color.Printf("Pulling %s \n",targetLocation)
			r,err := git.PlainOpen(targetLocation)
			if err != nil{
				color.Style{color.FgRed}.Printf("Failed to clone repo for %s Reason:%+v\n",repo.Name,err)
				//if we do it strict we should fail here!
				continue
			}
			err = r.Fetch(&git.FetchOptions{})
			if err != nil{
				color.Style{color.FgRed}.Printf("Failed to fetch repo for %s Reason:%+v\n",repo.Name,err)
			}
		}

	}


}

func (c *GoGitBackup) Check() error{
	repos := make([]Repository,0)

	for _, client := range c.clients {
		err := client.Init()
		if err != nil{
			color.Style{color.FgRed}.Printf("Failed to init client %s\n",client.Name())
			color.Style{color.FgBlack,color.BgGray}.Printf("Reason:%+v",err)
			return err
		}

		repo,err := client.List()
		if err != nil{
			color.Style{color.FgRed}.Printf("Failed to list repo for %s\n",client.Name())
			color.Style{color.FgBlack,color.BgGray}.Printf("Reason:%+v",err)
			return err
		}

		repos = append(repos,repo...)
	}

	c.repos = repos


	color.Style{color.FgBlack}.Printf("Found the following repositories:")
	fmt.Println("| Name\t| CreatedAt\t| Size [Byte?]\t|")
	for _, repo := range repos {
		fmt.Printf("| %s\t| %s\t| %d\t|\n",repo.Name,repo.CreatedAt,repo.Size)
	}

	return nil
}