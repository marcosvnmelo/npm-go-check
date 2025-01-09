package main

import (
	"fmt"
	"maps"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type PnpmWorkspace struct {
	Packages []string                `yaml:"packages"`
	Catalog  Dependencies            `yaml:"catalog"`
	Catalogs map[string]Dependencies `yaml:"catalogs"`
}

func (p PnpmWorkspace) GetPackages() Dependencies {
	packages := Dependencies{}

	maps.Copy(packages, p.Catalog)

	for _, catalog := range p.Catalogs {
		maps.Copy(packages, catalog)
	}
	return packages
}

func checkIfWorkspaceFileExists(entries []os.DirEntry) bool {
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if entry.Name() == "pnpm-workspace.yaml" {
			return true
		}
	}

	return false
}

func readPnpmWorkspace(filePath string) PnpmWorkspace {
	file, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error reading file")
		os.Exit(1)
	}

	var p PnpmWorkspace
	err = yaml.Unmarshal(file, &p)
	if err != nil {
		fmt.Println("Error parsing package.json")
		os.Exit(1)
	}

	return p
}

func isWorkspacePackage(version string) bool {
	return strings.HasPrefix(version, "workspace:")
}

func isCatalogPackage(version string) bool {
	return strings.HasPrefix(version, "catalog:")
}
