package blastradius

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/AlexsJones/kepler/commands/node"
)

func loadRepos(metarepo string) (map[string]node.PackageJSON, error) {
	repos := make(map[string]node.PackageJSON)

	repositories, err := ioutil.ReadDir(metarepo)
	if err != nil {
		return nil, err
	}

	for _, repo := range repositories {
		if repo.IsDir() && !strings.HasPrefix(repo.Name(), ".") {
			packageJSONfile := fmt.Sprintf("%s/%s/%s", metarepo, repo.Name(), "package.json")
			if _, err := os.Stat(packageJSONfile); err == nil {
				bytes, err := ioutil.ReadFile(packageJSONfile)
				if err != nil {
					return nil, fmt.Errorf("couldn't read package.json file in %s: %s", repo.Name(), err)
				}

				var tmp node.PackageJSON
				if err := json.Unmarshal(bytes, &tmp); err != nil {
					return nil, fmt.Errorf("couldn't read package.json file in %s: %s", repo.Name(), err)
				}

				repos[repo.Name()] = tmp
			}
		}
	}

	return repos, nil
}

// Calculate will identify other projects in the meta-repo that could be impacted by changes the given project
func Calculate(metarepo string, project string) ([]string, error) {
	repos, err := loadRepos(metarepo)
	if err != nil {
		return nil, err
	}

	blastRadius := make(map[string]map[string]bool)

	// In order to get all the packages that are used by the given project
	// it is needed to search through each package.json locally
	// to create a dep tree that way.
	for repo, pkg := range repos {
		for dep := range pkg.Dependencies {
			if _, exists := repos[dep]; exists {
				if _, exists := blastRadius[dep]; !exists {
					blastRadius[dep] = make(map[string]bool)
				}
				blastRadius[dep][repo] = true
			}
		}
	}

	var output []string

	for k := range blastRadius[project] {
		output = append(output, k)
	}

	return output, nil
}
