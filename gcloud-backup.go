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
	Readable bool
  Service string
  Account string
	Project string
	Region string
}

type Config struct {
  Service []string `json:"service"`
  Account string `json:"account"`
  Action struct {
		Import bool `json:"import,string"`
		Export bool `json:"export,string"`
	} `json:"action"`
	Project string `json:"projeoct"`
	Region string `json:"region"`
}

type Services struct {
	Export map[string]func(*jsonData)
}

type jsonData struct {
	Firewalls []*compute.Firewall `json:"firewalls,omitempty"`
	Routes []*compute.Route `json:"routes,omitempty"`
	Networks []*compute.Network `json:"networks,omitempty"`
	Addresses map[string][]*compute.Address `json:"addresses,omitempty"`
}

var flags = new(Flags)
var config = new(Config)
var services = new(Services)
var service *compute.Service
var output []byte

func init() {
	config.Action.Import = false
	config.Action.Export = false

	services.Export = make(map[string]func(*jsonData))
	services.Export["firewalls"] = exportFirewalls
	services.Export["routes"] = exportRoutes
	services.Export["networks"] = exportNetworks
	services.Export["addresses"] = exportAddresses

  flag.BoolVar(&flags.Version, "version", false, "Display version information")
  flag.BoolVar(&flags.Help, "help", false, "Display this help")
	flag.BoolVar(&flags.Export, "export", false, "Create new services export")
	flag.BoolVar(&flags.Import, "import", false, "Start services import from backup")
	flag.BoolVar(&flags.Readable, "readable", false, "Output JSON in readable format")
  flag.StringVar(&flags.Service, "service", "", "List of services to export/import (comma seperated)")
  flag.StringVar(&flags.Account, "account", "", "Google SDK account username")
	flag.StringVar(&flags.Project, "project", "", "Google SDK proect name")
	flag.StringVar(&flags.Region, "region", "", "Specify Google compute region")
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

	if flags.Region != "" {
    config.Region = flags.Region
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

func createServiceFromSDK() {
	log.Println("Creating new client from Google SDK config")
	sdk_config, err := google.NewSDKConfig(config.Account)
	if err != nil {
		log.Fatal(err)
	}

	client := sdk_config.Client(context.Background())
	service, err = compute.New(client)
	if err != nil {
		log.Fatal(err)
	}
}

func exportNetworks(exp *jsonData) {
	if service != nil {
		svc := service.Networks
	  list, err := svc.List(config.Project).Do()
		if err != nil {
	    log.Fatal(err)
	  } else {
			exp.Networks = list.Items
		}
	}
}

func exportAddresses(exp *jsonData) {
	if service != nil {
		exp.Addresses = make(map[string][]*compute.Address)
		svc := service.Addresses
	  list, err := svc.AggregatedList(config.Project).Do()
		if err != nil {
	    log.Fatal(err)
	  } else {
			for key, value := range list.Items {
				if len(value.Addresses) > 0 {
					exp.Addresses[key] = value.Addresses
				}
			}
		}
	}
}

func exportFirewalls(exp *jsonData) {
	if service != nil {
		svc := service.Firewalls
	  list, err := svc.List(config.Project).Do()
		if err != nil {
	    log.Fatal(err)
	  } else {
			exp.Firewalls = list.Items
		}
	}
}

func exportRoutes(exp *jsonData) {
	if service != nil {
		svc := service.Routes
	  list, err := svc.List(config.Project).Do()
		if err != nil {
	    log.Fatal(err)
	  } else {
			exp.Routes = list.Items
		}
	}
}

func main() {
	log.Printf("Google cloud backup - %v", version)
  if config.Action.Export {
		log.Printf("Starting export process of %v", config.Service)
		export := new(jsonData)
		createServiceFromSDK()
		for _, svc := range config.Service {
			if _, ok := services.Export[svc]; ok {
				services.Export[svc](export)
			} else {
				log.Printf("Error: invalid service - %v", svc)
			}
		}

		if flags.Readable {
			b, err := json.MarshalIndent(export, "", "  ")
			if err != nil {
				log.Fatal(err)
			}
			output = b
		} else {
			b, err := json.Marshal(export)
			if err != nil {
				log.Fatal(err)
			}
			output = b
		}
		os.Stdout.Write(output)
	} else if config.Action.Import {
		log.Printf("Starting import process of %v", config.Service)
	}
}
