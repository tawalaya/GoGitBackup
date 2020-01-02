package backup

import (
	"fmt"
	"github.com/xanzy/go-gitlab"
	"strings"
)

type _gitlabClient struct {
	Token   string
	BaseURL string
	client  *gitlab.Client
	name    string
}

func (c *_gitlabClient) Init() error {

	git := gitlab.NewClient(nil, c.Token)
	if c.BaseURL != "" {
		err := git.SetBaseURL(c.BaseURL)
		if err != nil {
			log.Debug("failed to set base url to %s, %+v", c.BaseURL, err)
			return err
		}

	}
	c.client = git

	return nil
}

func (c *_gitlabClient) List() ([]Repository, error) {
	projects, res, err := c.client.Projects.ListProjects(&gitlab.ListProjectsOptions{})

	if err != nil {
		log.Debug("failed to list GitHub repositories reason %+v", res)
		return nil, err
	}

	repoList := make([]Repository, 0)
	for _, project := range projects {
		var size int64
		if project.Statistics != nil {
			size = project.Statistics.StorageSize
		} else {
			size = -1
		}

		repoList = append(repoList, Repository{
			CloneUrl:  strings.Replace(project.HTTPURLToRepo, "https://", fmt.Sprintf("https://oauth2:%s@", c.Token), -1),
			Name:      strings.ReplaceAll(strings.ReplaceAll(project.NameWithNamespace, " / ", "/"), " ", "_"),
			Size:      size,
			CreatedAt: *project.CreatedAt,
		})
	}

	return repoList, nil
}

func (c *_gitlabClient) Name() string {
	return c.name
}
