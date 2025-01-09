package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	PACKAGE_ARGUMENTS = [2]string{"-p", "--package"}
	UPDATE_ARGUMENTS  = [2]string{"-u", "--update"}
	HELP_ARGUMENTS    = [2]string{"-h", "--help"}
)

var (
	packagesCache      = map[string]Package{}
	packagesCacheMutex = sync.Mutex{}
	packagesWg         = sync.WaitGroup{}
)

var dependenciesCache = map[string]Dependencies{}

var pnpmDependencies = Dependencies{}

func main() {
	config := struct {
		Packages []string
		Update   bool
	}{
		Packages: []string{},
		Update:   false,
	}

	for i, arg := range os.Args[1:] {
		switch arg {
		case PACKAGE_ARGUMENTS[0], PACKAGE_ARGUMENTS[1]:
			valueArg := os.Args[i+2]

			if valueArg == "" {
				fmt.Println("Missing package name")
				os.Exit(1)
			}

			if strings.HasPrefix(valueArg, "-") {
				fmt.Println("Invalid package name:", valueArg)
				os.Exit(1)
			}

			config.Packages = append(config.Packages, valueArg)
		case UPDATE_ARGUMENTS[0], UPDATE_ARGUMENTS[1]:
			config.Update = true
		case HELP_ARGUMENTS[0], HELP_ARGUMENTS[1]:
			fmt.Println("Usage: npm-go-check [options]")
			fmt.Println("Options:")
			fmt.Println("  -p, --package <name>  Check the latest version of a package")
			fmt.Println("  -u, --update          Update all packages")
			os.Exit(0)
		default:
			if i > 0 {
				prevArg := os.Args[i-1]

				if PACKAGE_ARGUMENTS[0] == prevArg || PACKAGE_ARGUMENTS[1] == prevArg {
					continue
				}
			}
		}
	}

	if len(config.Packages) > 0 {
		fetchPackagesFromCli(config.Packages)
		return
	}

	fetchPackagesFromDir(config.Update)
}
