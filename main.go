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
	"github.com/sirupsen/logrus"
	lib "github.com/tawalaya/GoGitBackup/backup"
	"github.com/urfave/cli/v2"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
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
	logger.SetLevel(logrus.InfoLevel)
	log = logger.WithFields(logrus.Fields{
		"prefix": "git-backup",
		"build":  Build,
	})
}

func main() {
	app := &cli.App{
		Name:  "gitback",
		Usage: "Utility to backup your git(Hub|lab) accounts.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Load configuration from `FILE`",
				Value:   "./config.yml",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enables verbose logging",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "backup",
				Aliases: []string{"b"},
				Usage:   "performs backup of all git(hub/lab) accounts that can be accessed.",
				Action: func(c *cli.Context) error {
					client := preflight(c)
					client.Do()
					return nil
				},
			},
			{
				Name:    "check",
				Aliases: []string{"c"},
				Usage:   "check what we can backup using this utility and also validates your config ;)",
				Action: func(c *cli.Context) error {
					client := preflight(c)
					return client.Check()
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func preflight(c *cli.Context) *lib.GoGitBackup {

	bytes, err := ioutil.ReadFile(c.String("config"))

	if err != nil {
		log.Fatalf("failed to read config at %s %+v", c.String("config"), err)
	}

	var config lib.Config

	err = yaml.Unmarshal(bytes, &config)

	if err != nil {
		log.Fatalf("failed to parse config, %+v", err)
	}

	if c.Bool("verbose") {
		logger.SetLevel(logrus.DebugLevel)
		log.Debugf("using config:\n %+v", config)
	} else {
		logger.SetLevel(logrus.ErrorLevel)
	}
	client, err := lib.NewGoBackup(&config)

	lib.SetLogger(logger)
	lib.SetLog(log)

	if err != nil {
		log.Fatalf("failed to create backup client due to %+v", err)
	}

	return client

}
