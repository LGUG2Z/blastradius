package main

import (
	"github.com/LGUG2Z/blastradius/blastradius"
	"github.com/fatih/color"
	"os"
)

func main() {
	projects, err := blastradius.RunTestsOn(os.Args[1], "yarn", "test")
	if err != nil {
		color.Red("Could not load project: %v", err)
		os.Exit(1)
	}

	combinedStatus := 0
	for result := range projects {
		combinedStatus += result.ExitCode
		switch result.ExitCode {
		// Good state
		case 0:
			color.Green("Test pass: %v\n", result.Name)
		// bad state
		default:
			color.Red("Test failed: %v, see:\n%v\n", result.Name, string(result.Output))
		}
	}

	os.Exit(combinedStatus)
}
