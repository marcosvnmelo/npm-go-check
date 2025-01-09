package main

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	ignore "github.com/sabhiram/go-gitignore"
)

const NPM_BASE_URL = "https://registry.npmjs.org/"

type DistTags struct {
	Next   string `json:"next"`
	Latest string `json:"latest"`
}

type Package struct {
	Name     string   `json:"name"`
	DistTags DistTags `json:"dist-tags"`
}

type Dependencies map[string]string

type PackageJson struct {
	Name                 string       `json:"name"`
	Dependencies         Dependencies `json:"dependencies"`
	DevDependencies      Dependencies `json:"devDependencies"`
	OptionalDependencies Dependencies `json:"optionalDependencies"`
	PeerDependencies     Dependencies `json:"peerDependencies"`
}

func (p Package) GetLatestVersion() string {
	return p.DistTags.Latest
}

func fetchPackage(name string, wg *sync.WaitGroup) {
	packagesCacheMutex.Lock()
	if _, ok := packagesCache[name]; ok {
		packagesCacheMutex.Unlock()
		return
	}
	packagesCacheMutex.Unlock()

	url := NPM_BASE_URL + name

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching package")
		os.Exit(1)
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		fmt.Printf("Package not found: %s\n", name)
		os.Exit(1)
	}

	if resp.StatusCode != 200 {
		fmt.Println("Error fetching package")
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error fetching package")
		os.Exit(1)
	}

	p := Package{}
	err = json.Unmarshal(body, &p)
	if err != nil {
		fmt.Println("Error fetching package")
		os.Exit(1)
	}

	packagesCacheMutex.Lock()
	packagesCache[name] = p
	packagesCacheMutex.Unlock()
	wg.Done()
}

func fetchPackagesFromCli(cliPackages []string) {
	for _, name := range cliPackages {
		packagesWg.Add(1)
		go fetchPackage(name, &packagesWg)
	}

	packagesWg.Wait()

	for _, p := range cliPackages {
		p := packagesCache[p]
		fmt.Println(p.Name, p.GetLatestVersion())
	}
}

func fetchPackagesFromDir(update bool) {
	filePaths := []string{}

	readDirectories(".", &filePaths)

	if len(filePaths) == 0 {
		fmt.Println("No package.json files found")
		os.Exit(0)
	}

	for _, filePath := range filePaths {
		file, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Println("Error reading file")
			os.Exit(1)
		}

		fmt.Println(filePath)

		var p PackageJson
		err = json.Unmarshal(file, &p)
		if err != nil {
			fmt.Println("Error parsing package.json")
			os.Exit(1)
		}

		fileDependencies := Dependencies{}

		maps.Copy(fileDependencies, p.Dependencies)
		maps.Copy(fileDependencies, p.DevDependencies)
		maps.Copy(fileDependencies, p.OptionalDependencies)
		maps.Copy(fileDependencies, p.PeerDependencies)

		dependenciesCache[filePath] = fileDependencies
	}

	allDependencies := Dependencies{}
	for _, fileDependencies := range dependenciesCache {
		maps.Copy(allDependencies, fileDependencies)
	}

	fmt.Printf("Fetching dependencies...\n\n")
	fetchDependencies(allDependencies)

	for index, filePath := range filePaths {
		if index > 0 {
			fmt.Println()
		}

		fmt.Printf("File: %s\n", filePath)
		printDependencies(dependenciesCache[filePath])
	}

	if update {
		fmt.Println("Updating packages")
	}
}

func readDirectories(dir string, filePaths *[]string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("Error reading directory")
		os.Exit(1)
	}

	ignoreObject, err := ignore.CompileIgnoreFile(dir + "/.gitignore")
	hasIgnoreFile := err == nil

	for _, entry := range entries {

		if hasIgnoreFile && ignoreObject.MatchesPath(entry.Name()) {
			continue
		}

		fullName := dir + "/" + entry.Name()

		if entry.IsDir() {
			if entry.Name() == ".git" {
				continue
			}

			if entry.Name() == "node_modules" {
				continue
			}

			readDirectories(fullName, filePaths)
			continue
		}

		if entry.Name() == "package.json" {
			*filePaths = append(*filePaths, fullName)
		}
	}

	if dir == "." && checkIfWorkspaceFileExists(entries) {
		pnpmWorkspace := readPnpmWorkspace("pnpm-workspace.yaml")
		pnpmDependencies = pnpmWorkspace.GetPackages()
	}
}

func fetchDependencies(dependencies Dependencies) {
	for name, version := range dependencies {
		if isWorkspacePackage(version) {
			continue
		}

		packagesWg.Add(1)
		go fetchPackage(name, &packagesWg)
	}
	packagesWg.Wait()
}

func printDependencies(dependencies Dependencies) {
	if len(dependencies) == 0 {
		fmt.Println(" No dependencies found")
		return
	}

	entriesIter := maps.All(dependencies)
	entries := make([][2]string, 0, len(dependencies))

	for key, value := range entriesIter {
		entries = append(entries, [2]string{key, value})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i][0] < entries[j][0]
	})

	biggestDependencyNameLength := 0
	biggestDependencyOriginalVersionLength := 0
	biggestDependencyLatestVersionLength := 0

	for name, version := range entriesIter {
		if isWorkspacePackage(version) {
			continue
		}

		if isCatalogPackage(version) {
			version = pnpmDependencies[name]
		}

		if len(name) > biggestDependencyNameLength {
			biggestDependencyNameLength = len(name)
		}
		if len(version) > biggestDependencyOriginalVersionLength {
			biggestDependencyOriginalVersionLength = len(version)
		}

		npmPackage, ok := packagesCache[name]

		if !ok {
			continue
		}

		latestVersion := npmPackage.GetLatestVersion()
		firstChar := version[0:1]
		if _, err := strconv.ParseInt(firstChar, 10, 32); err != nil {
			latestVersion = firstChar + latestVersion
		}

		if len(latestVersion) > biggestDependencyLatestVersionLength {
			biggestDependencyLatestVersionLength = len(latestVersion)
		}
	}

	printCount := 0
	for _, entry := range entries {
		name := entry[0]
		version := entry[1]

		if isWorkspacePackage(version) {
			continue
		}

		if isCatalogPackage(version) {
			version = pnpmDependencies[name]
		}

		npmPackage, ok := packagesCache[name]

		if !ok {
			fmt.Println("Package not found:", name)
			continue
		}

		latestVersion := npmPackage.GetLatestVersion()

		firstChar := version[0:1]
		var versionSignal string
		if _, err := strconv.ParseInt(firstChar, 10, 32); err == nil {
			if latestVersion == version {
				continue
			}
		} else {
			versionSignal = firstChar
			numberVersion := version[1:]
			if latestVersion == numberVersion {
				continue
			}
		}

		if versionSignal != "" {
			latestVersion = versionSignal + latestVersion
		}

		nameToVersionSpaces := strings.Repeat(" ", biggestDependencyNameLength-len(name))
		originalVersionSpaces := strings.Repeat(" ", biggestDependencyOriginalVersionLength-len(version))
		latestVersionSpaces := strings.Repeat(" ", biggestDependencyLatestVersionLength-len(latestVersion))

		catalogSignal := ""

		if isCatalogPackage(dependencies[name]) {
			catalogSignal = "(catalog)"
		}

		fmt.Printf(" %s%s%s  %s  →  %s%s %s\n",
			name,
			nameToVersionSpaces,
			originalVersionSpaces,
			version,
			latestVersionSpaces,
			latestVersion,
			catalogSignal,
		)
		printCount++
	}

	if printCount == 0 {
		fmt.Println(" No dependencies to update")
	}
}
