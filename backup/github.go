package backup

import (
	"context"
	"fmt"

	"strings"

	"github.com/d5/tengo/v2"
	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
)

type _githubClient struct {
	ctx     context.Context
	client  *github.Client
	Token   string
	User    string
	name    string
	filters []*tengo.Script
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

func (c *_githubClient) RegisterFilter(filters []*tengo.Script) {
	c.filters = filters
}

func (c *_githubClient) List() ([]Repository, error) {
	repoList := make([]Repository, 0)

	search := &github.RepositoryListOptions{
		Visibility: "all",
		//Affiliation: "owner",
		//Type:        "all",
		Sort: "created",
		ListOptions: github.ListOptions{
			PerPage: 50,
			Page:    0,
		},
	}

	for {
		list, res, err := c.client.Repositories.List(c.ctx, "", search)

		if err != nil {
			log.Debugf("failed to list GitHub repositories reason %+v", res)
			return nil, err
		}

		for _, repo := range list {
			log.Debugf("got %s size %d", repo.GetFullName(), repo.GetSize())

			url := repo.GetCloneURL()

			if c.User != "" {
				url = strings.Replace(url, "https://", fmt.Sprintf("https://%s:%s@", c.User, c.Token), -1)
			}

			visibility := Public
			if repo.Private != nil && *repo.Private {
				visibility = Private
			}

			owner := repo != nil && repo.Owner != nil && repo.Owner.Name != nil && *repo.Owner.Name == c.User

			archived := repo != nil && repo.Archived != nil && *repo.Archived
			r := Repository{
				CloneUrl:     url,
				Name:         repo.GetFullName(),
				Size:         int64(repo.GetSize()),
				CreatedAt:    repo.GetCreatedAt().UTC(),
				Owner:        owner,
				Member:       true,
				Visibility:   visibility,
				Archived:     archived,
				ProviderName: c.name,
			}

			if filter(r, c.filters) {
				repoList = append(repoList, r)
			}

		}

		//check if there are more pages...
		if res.NextPage == 0 {
			break
		}

		search.Page = res.NextPage
	}
	return repoList, nil
}
