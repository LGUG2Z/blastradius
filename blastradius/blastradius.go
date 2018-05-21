package blastradius

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"errors"
	nodejs "github.com/AlexsJones/kepler/commands/node"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
)

func loadRepos(metarepo string) (map[string]nodejs.PackageJSON, error) {
	repos := make(map[string]nodejs.PackageJSON)

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

				var tmp nodejs.PackageJSON
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

	if _, exists := repos[project]; !exists {
		return nil, fmt.Errorf("%s not found", project)
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

// TestedProject contains all the data required to
// send back to main process and report weather or not it failed
type TestedProject struct {
	Name     string
	ExitCode int
	Output   []byte
}

// RunTestsOn will test the given project
// and all projects that use the given project
func RunTestsOn(project string, command ...string) (chan TestedProject, error) {
	projects, err := Calculate(".", project)
	if err != nil {
		return nil, err
	}
	results := make(chan TestedProject, runtime.NumCPU())
	wg := sync.WaitGroup{}
	// detaching the dispatcher thread from the main as
	// as to avoid blocking the main thread
	go func(ch chan TestedProject, projects []string) {
		wg.Add(len(projects))
		for _, p := range projects {
			go func(p string, wg *sync.WaitGroup, ch chan TestedProject) {
				ret, err := executeTests(p, "npm", "test")
				if err != nil {
					// Not sure what to do here
				}
				ch <- ret
				wg.Done()
			}(p, &wg, ch)
		}
		wg.Wait()
		close(ch)
	}(results, append(projects, project))
	return results, nil
}

func executeTests(project string, cmd ...string) (TestedProject, error) {
	if len(cmd) < 2 {
		return TestedProject{}, errors.New("Not enough arguments passed for command")
	}
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Dir = project
	buff, err := c.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exiter, ok := err.(*exec.ExitError); ok {
			if status, ok := exiter.Sys().(syscall.WaitStatus); ok {
				exitCode = int(status.ExitStatus())
			}
		}
	}
	return TestedProject{
		Name:     project,
		ExitCode: exitCode,
		Output:   buff,
	}, nil
}
