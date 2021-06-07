# GoGitBackup

This is a utility to download and backup all your GitHub/GitLab accounts to disk.
It uses the APIs of each provider to find all the repositories you have access to and clones or pulls them into the same root directory. 

This tool is intended for local backups of all __your__ work. The tool uses the git-provider API to list all repositories you have access to and clones them. It is intended to backup work from internal Github/GitLab repositories in case you lose access to them, e.g., in case you leave university.

## Usage
The tool is based on a `*.yml` config file where you can define one or more accounts to back up.

```
NAME:
   GoGitBackup - Utility to backup your git(Hub|lab) accounts.

USAGE:
   GoGitBackup [global options] command [command options] [arguments...]

COMMANDS:
   backup, b  performs a backup of all git(hub/lab) accounts that can be accessed.
   check, c   check what we can backup using this utility and also validates your config ;)
   help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config FILE, -c FILE  Load configuration from FILE (default: "./config.yml")
   --verbose, -v           Enables verbose logging (default: false)
   --help, -h              show help (default: false)

```

### Config
To run the utility, you need to specify at least one account and a local repository. 
An exemplary config file can look like this:
```yml
repository: /tmp/test
accounts:
  - name: Personal GitHub
    token: < oauht token >
    provider: 0
    args:
      - tawalaya
```
The following accounts are supported:

#### GitHub
For GitHub, you need to specify the following fields in the config file:
```yml
  - name: <A Name of this account for logging>
    token: <a github access token>
    provider: 0
    args:
      - <the github usename of the token>
```

In order to obtain a GitHub token, follow this guide  [guid](https://docs.github.com/en/github/authenticating-to-github/keeping-your-account-and-data-secure/creating-a-personal-access-token).
 
 #### GitLab
 For GitLab, you need to specify the following fields in the config file:
```yml
     - name: <A Name of this account for logging>
       token: <Gitlab API Token>
       provider: 1
       args:
         - <base-url of your GitLab installation, optional will use gitlab.com by default>
 ```
 
 In order to obtain a GitLab token, follow this [guid](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html). 

### Filters
Sometimes you want or need to avoid some repositories. For that GoGitBackup can add filters.
Each filter is added to the `filters` property of each provider config. 
For the filter implementation we use [Tengo](github.com/d5/tengo/v2) Script. Each filter is executed in sequence. Each filter can use the following variables.

| name | description |
| ---- | ----------- |
| owner | bool - true if the reposetory is owned by the git user. | 
| member| bool - true if the git user is a member of reposetory. |
| visibility | int - Public = 0, Private = 1, Internal = 2 | 
| size | int - size of the reposetory | 
| name | string - name of the reposetory |

Each script needs to set a variable `r`; for example, `r := owner` checks if the repository is owned by the token owner. 
```yml
     - name: <A Name of this account for logging>
       token: <Gitlab API Token>
       provider: 1
       args:
         - <base-url of your GitLab installation, optional will use gitlab.com by default>
       filters: 
         - "r:=owner"
 ```

## Development
The tool is based on go-lang 1.13 and should be easily extendable to other git-as-a-service providers, PR's welcome.