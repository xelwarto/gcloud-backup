/*
Copyright 2016 Ted Elwartowski <xelwarto.pub@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
  "os"
  "fmt"
  "strings"
	"log"

	"encoding/json"
	"golang.org/x/net/context"
  "golang.org/x/oauth2/google"
  "google.golang.org/api/compute/v1"
)

var version = string("v0.1.0")

type Flags struct {
  Version bool
  Help bool
	Import bool
	Export bool
  Service string
  Account string
	Project string
}

type Config struct {
  Service []string `json:"service"`
  Account string `json:"account"`
  Action struct {
		Import bool `json:"import,string"`
		Export bool `json:"export,string"`
	} `json:"action"`
	Project string `json:"projeoct"`
	ComputeService *compute.Service `json:"-"`
}

type Services struct {
	Export map[string]func(*jsonData)
}

type jsonData struct {
	Firewalls []*compute.Firewall `json:"firewalls,omitempty"`
}

var flags = new(Flags)
var config = new(Config)
var services = new(Services)

func init() {
	config.Action.Import = false
	config.Action.Export = false

	services.Export = make(map[string]func(*jsonData))
	services.Export["firewalls"] = exportFirewalls

  flag.BoolVar(&flags.Version, "version", false, "Display version information")
  flag.BoolVar(&flags.Help, "help", false, "Display this help")
	flag.BoolVar(&flags.Export, "export", false, "Create new services export")
	flag.BoolVar(&flags.Import, "import", false, "Start services import from backup")
  flag.StringVar(&flags.Service, "service", "", "List of services to export/import (comma seperated)")
  flag.StringVar(&flags.Account, "account", "", "Google SDK account username")
	flag.StringVar(&flags.Project, "project", "", "Google SDK proect name")
  flag.Parse()

  if flags.Version {
    fmt.Fprintf(os.Stderr, "Version: %v\n", version)
    os.Exit(1)
  }

  if flags.Help {
    showUsage()
  }

  if flags.Service == "" {
    showUsage("please include a service")
  } else {
    // ERROR HANDLING
    config.Service = strings.Split(flags.Service, ",")
  }

  if flags.Account == "" {
    showUsage("please specify a Google SDK user account")
  } else {
    config.Account = flags.Account
  }

	if flags.Project == "" {
    showUsage("please specify a Google SDK project")
  } else {
    config.Project = flags.Project
  }

	if flags.Import || flags.Export {
		if flags.Import && flags.Export {
			showUsage("please select an action - export/import")
		} else if flags.Export {
			config.Action.Export = true
		} else if flags.Import {
			config.Action.Import = true
			showUsage("import action not implmented")
		}
	} else {
		showUsage("please select an action - export/import")
	}
}

func showUsage(s ...string) {
  if len(s) > 0 && len(s[0]) > 0 {
    fmt.Fprintf(os.Stderr, "Error: %v\n\n", s[0])
  }

  fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  gcloud-backup [-import|-export] -account=<user_name> -project=<project_name> -services=<service_list> OPTIONS...\n\n")
	flag.PrintDefaults()
  os.Exit(1)
}

func createComputeService() {
	log.Println("Creating new client from Google SDK config")
	sdk_config, err := google.NewSDKConfig(config.Account)
	if err != nil {
		log.Fatal(err)
	}

	client := sdk_config.Client(context.Background())
	config.ComputeService, err = compute.New(client)
	if err != nil {
		log.Fatal(err)
	}
}

func exportFirewalls(exp *jsonData) {
	if config.ComputeService != nil {
		fw_svc := compute.NewFirewallsService(config.ComputeService)
	  fw_list, err := fw_svc.List(config.Project).Do()
		if err != nil {
	    log.Fatal(err)
	  } else {
			exp.Firewalls = fw_list.Items
		}
	}
}

func main() {
	log.Printf("Google cloud backup - %v", version)
  if config.Action.Export {
		log.Printf("Starting export process of %v", config.Service)
		export := new(jsonData)
		createComputeService()
		for _, svc := range config.Service {
			if _, ok := services.Export[svc]; ok {
				services.Export[svc](export)
			} else {
				log.Printf("Error: invalid service - %v", svc)
			}
		}
		//output, err := json.Marshal(export)
		output, err := json.MarshalIndent(export, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		os.Stdout.Write(output)

	} else if config.Action.Import {
		log.Printf("Starting import process of %v", config.Service)
	}
}
