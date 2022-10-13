package main

import (
	"fmt"
	"encoding/json"
	"io/ioutil"
	"os"
)

type TemplateStructure struct {
	Inherit									bool			`json:"inherit"`
	ParentTemplate					string		`json:"parent"`
	Git											bool			`json:"git"`	
	Gitignore								string		`json:"gitignore"`
	Env											bool			`json:"env"`
	ExternalProgramsEnd			[]string	`json:"externalProgramsEnd"`
	ExternalProgramsStart		[]string	`json:"externalProgramsStart"`
	Dirs										[]string	`json:"dirs"`
	Files										[]string	`json:"files"`
}

type Template map[string]TemplateStructure

func ReadConfig() Template {
	file, _ := os.UserConfigDir()
	file += "/creo/config.json"

	reader, err := os.Open(file)
	if err != nil {
		fmt.Println("Config file not found")
		os.Exit(1)
	}

	byteData, _ := ioutil.ReadAll(reader)
	defer reader.Close()
	
	Project := Template{}
	json := json.Unmarshal(byteData, &Project)
	if json != nil {
		os.Exit(1)
	}

	return Project
}

func main() {
	testProject := ReadConfig()	
	fmt.Println(testProject)
}
