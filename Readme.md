# GoGitBackup

This is a utility to download and backup all your GitHub/GitLab accounts to disk.
It uses the APIs of each provider to find all the reposeories you have access to and clones or pulls them into the same root directory. 

This tool is intended for local backups of all __your__ work. The tool uses the git-provider API to list all repositories you have access to and clones them. It is intended to backup work from internal Github/GitLab repositories in case you lose access to them, e.g., in case you leave university.

## Usage
The tool is based of a `*.yml` config file where you can define one or more accounts to back up.

```
NAME:
   gitback - Utility to backup your git(Hub|lab) accounts.

USAGE:
   backup-git [global options] command [command options] [arguments...]

COMMANDS:
   backup, b  performs backup of all git(hub/lab) accounts that can be accessed.
   check, c   check what we can backup using this utility and also validates your config ;)
   help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config FILE, -c FILE  Load configuration from FILE (default: "./config.yml")
   --verbose, -v           Enables verbose logging (default: false)
   --help, -h              show help (default: false)

```

### Config
In order to run the utily, you need to specify at least one accont and a local reposetory. 
An exemplary config file can look like this:
```
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
```
  - name: <A Name of this account for logging>
    token: <a github access token>
    provider: 0
    args:
      - <the github usename of the token>
```

Inorder to obtain a github token follow the this guid ()[]. 
#### Filters
Sometimes you want or need to avoid some reposetories, for that GoGitBackup has the ability to add filters.
Each filter is added to the `filterList` propertie to the config. For the filter implementation we use (Tengo)[github.com/d5/tengo/v2] Script. Each filter is executed in sequence, each filter can use the following variabels.

| name | description |
| ---- | ----------- |
| owner | bool - true if the reposetory is owned by the git user. | 
| member| bool - true if the git user is a member of reposetory. |
| visibility | int - Public = 0, Private = 1, Internal = 2 | 
| size | int - size of the reposetory | 
| name | string - name of the reposetory |

Each script needs to set a variable `r`, for example, `r := owner` checks if the resposetory is owned by the token owner. 
##### Planed Features

 - [ ] Store all Issues
 - [ ] Store the Wiki
 - [ ] Store all releases
 
 #### GitLab
 For GitLab, you need to specify the following fields in the config file:
 ```
     - name: <A Name of this account for logging>
       token: <Gitlab API Token>
       provider: 1
       args:
         - <base-url of your gitlab installation, optional will use gitlab.com by default>
 ```
 
 Inorder to obtain a gitlab token follow the this guid ()[]. 
 
##### Planed Features

 - [ ] Store all Issues
 - [ ] Store the Wiki

## Development
The tool is based on go-lang 1.13 and should be easily extendable to other git-as-a-service providers, PR's welcome.
 
### Planed Features
 - [ ] Enable different git-authentication methods
 - [ ] Enable Error Resolution for failing pulls (clone/overwrite)
 - [x] Improve the output text to be more helpful
 - [x] Add a progress bar

