package backup

import (
	"context"
	"fmt"
	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"

	"strings"
)

type _githubClient struct {
	ctx    context.Context
	client *github.Client
	Token  string
	User   string
	name   string
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

func (c *_githubClient) List() ([]Repository, error) {

	list, res, err := c.client.Repositories.List(c.ctx, "", &github.RepositoryListOptions{
		Visibility: "all",
		//Affiliation: "owner",
		//Type:        "all",
		Sort: "created",
	})

	if err != nil {
		log.Debug("failed to list GitHub repositories reason %+v", res)
		return nil, err
	}

	repoList := make([]Repository, 0)

	for _, repo := range list {
		log.Debugf("got %s size %d", repo.GetFullName(), repo.GetSize())

		url := repo.GetCloneURL()

		if c.User != "" {
			url = strings.Replace(url, "https://", fmt.Sprintf("https://%s:%s@", c.User, c.Token), -1)
		}

		repoList = append(repoList,
			Repository{
				CloneUrl:  url,
				Name:      repo.GetFullName(),
				Size:      int64(repo.GetSize()),
				CreatedAt: repo.GetCreatedAt().UTC(),
			})

	}

	return repoList, nil
}
