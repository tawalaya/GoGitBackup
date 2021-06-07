package backup

import (
	"fmt"
	"strings"

	"github.com/xanzy/go-gitlab"
	"github.com/d5/tengo/v2"
)

type _gitlabClient struct {
	Token   string
	BaseURL string
	client  *gitlab.Client
	name    string
	user    *gitlab.User
	filters []*tengo.Script
}

func (c *_gitlabClient) Init() error {

	git := gitlab.NewClient(nil, c.Token)
	if c.BaseURL != "" {
		err := git.SetBaseURL(c.BaseURL)
		if err != nil {
			log.Debugf("failed to set base url to %s, %+v", c.BaseURL, err)
			return err
		}

	}
	c.client = git

	user, _, err := git.Users.CurrentUser()
	if err != nil {
		return err
	}

	c.user = user

	return nil
}

func (c *_gitlabClient) List() ([]Repository, error) {
	projects, res, err := c.client.Projects.ListProjects(&gitlab.ListProjectsOptions{})

	if err != nil {
		log.Debugf("failed to list GitHub repositories reason %+v", res)
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

		isMember := false
		members, _, err := c.client.ProjectMembers.ListAllProjectMembers(project.ID, nil)
		if err == nil {
			for _, m := range members {
				if m.ID == c.user.ID {
					isMember = true
					break
				}
			}
		}

		visibility := Private
		switch project.Visibility {
		case gitlab.PrivateVisibility:
			visibility = Private
			break
		case gitlab.InternalVisibility:
			visibility = Internal
			break
		case gitlab.PublicVisibility:
			visibility = Public
			break
		}

		r := Repository{
			CloneUrl:   strings.Replace(project.HTTPURLToRepo, "https://", fmt.Sprintf("https://oauth2:%s@", c.Token), -1),
			Name:       strings.ReplaceAll(strings.ReplaceAll(project.NameWithNamespace, " / ", "/"), " ", "_"),
			Size:       size,
			CreatedAt:  *project.CreatedAt,
			Owner:      project.Owner != nil && project.Owner.ID == c.user.ID,
			Member:     isMember,
			Visibility: visibility,
		}

		if filter(r,c.filters) {
			repoList = append(repoList,r)
		}
	}

	return repoList, nil
}

func (c *_gitlabClient) Name() string {
	return c.name
}
func (c *_gitlabClient) RegisterFilter(filters []*tengo.Script)  {
	c.filters = filters
}