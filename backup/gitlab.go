package backup

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/xanzy/go-gitlab"
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

	ops := make([]gitlab.ClientOptionFunc, 0)
	if c.BaseURL != "" {
		ops = append(ops, gitlab.WithBaseURL(c.BaseURL))
	}

	git, err := gitlab.NewClient(c.Token, ops...)

	if err != nil {
		log.Debugf("failed to create client for %s, %+v", c.BaseURL, err)
		return err
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
	//grep all active projects
	opt := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20,
			Page:    1,
		},
		Membership: gitlab.Bool(true),
		Statistics: gitlab.Bool(true),
	}

	list, err := c.list(opt)

	if err != nil {
		return nil, err
	}

	//enable search of archived projects
	opt.Page = 1
	opt.Archived = gitlab.Bool(true)

	archived, err := c.list(opt)

	if err != nil {
		return nil, err
	}

	return append(list, archived...), nil
}

func (c *_gitlabClient) list(opt *gitlab.ListProjectsOptions) ([]Repository, error) {
	repoList := make([]Repository, 0)

	for {
		projects, resp, err := c.client.Projects.ListProjects(opt)

		if err != nil {
			log.Debugf("failed to list GitHub repositories reason %+v", resp)
			return nil, err
		}

		for _, project := range projects {
			log.Debugf("got %s", project.Name)

			r := c.generate(project)

			if filter(r, c.filters) {
				repoList = append(repoList, r)
			}
		}

		if resp.CurrentPage >= resp.TotalPages {
			return repoList, nil
		}

		// Update the page number to get the next page.
		opt.Page = resp.NextPage
	}
}

func (c *_gitlabClient) generate(project *gitlab.Project) Repository {

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
		CloneUrl:     strings.Replace(project.HTTPURLToRepo, "https://", fmt.Sprintf("https://oauth2:%s@", c.Token), -1),
		Name:         strings.ReplaceAll(strings.ReplaceAll(project.NameWithNamespace, " / ", "/"), " ", "_"),
		Size:         size,
		CreatedAt:    *project.CreatedAt,
		Owner:        project.Owner != nil && project.Owner.ID == c.user.ID,
		Member:       isMember,
		Visibility:   visibility,
		Archived:     project.Archived,
		ProviderName: c.name,
	}

	log.Debugf("got %s %+v %+v %+v", r.Name, r.Member, r.Owner, r.Size)
	return r
}

func (c *_gitlabClient) Name() string {
	return c.name
}
func (c *_gitlabClient) RegisterFilter(filters []*tengo.Script) {
	c.filters = filters
}
