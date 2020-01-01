# Backup Git

This is a utility to download and backup all your GitHub/GitLab accounts to disk.
It uses the APIs of each provider to find all the reposeories you have access to and clones or pulls them into the same root directory. 

This tool is intended for local backups of all __your__ work.

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

### GitHub
For GitHub, you need to specify the following fields in the config file:
```
  - name: <A Name of this account for logging>
    token: <a github access token>
    provider: 0
    args:
      - <the github usename of the token>
```

Inorder to obtain a github token follow the this guid ()[]. 

#### Planed Features

 - [X] Pull private and public repos of a user
 - [ ] Add flag to only select public repos
 - [ ] Add flag to only select repos the user owns
 - [ ] Store all Issues
 - [ ] Store the Wiki
 - [ ] Store all releases
 
 ### GitLab
 For GitLab, you need to specify the following fields in the config file:
 ```
     - name: <A Name of this account for logging>
       token: <Gitlab API Token>
       provider: 1
       args:
         - <base-url of your gitlab installation, optional will use gitlab.com by default>
 ```
 
 Inorder to obtain a gitlab token follow the this guid ()[]. 
 
#### Planed Features

 - [X] Pull private and public repos of a user
 - [ ] Add a filter for what to select
 - [ ] Store all Issues
 - [ ] Store the Wiki

## Development
The tool is based on go-lang 1.13 and should be easily extendable to other git-as-a-service providers, PR's welcome.
 
#### Planed Features
 - [ ] Enable differnt git-authentication methods
 - [ ] Enable Error Resolution for failing pulls (clone/overwrite)
 - [ ] Improve the output text to be more helpfull
 - [ ] Add a progress bar

