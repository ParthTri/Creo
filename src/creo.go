package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
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

type Project struct {
	Name				string
	Path				string
	Structure		*TemplateStructure
}

// TODO: Allow for user to set the path to their Projects directory or use the one supplied by the project structure
func NewProject(name string, structure *TemplateStructure) Project {
	project := Project{
		Name:				name,
		Structure: structure,
	}

	project.Path, _ = os.UserHomeDir() 
	project.Path += "/Projects/" + project.Name

	return project
}
// Initiate Git and append data to gitignore file
func (project Project)Git() error {
	cmd := exec.Command("git", "init", project.Path)
	err := cmd.Run()
	if err != nil {
		return err
	}

	gitignore := fmt.Sprintf("%v/.gitignore", project.Path)
	err = os.WriteFile(gitignore, []byte(project.Structure.Gitignore), 0666)
	return err
}
func input(prompt string, reader *bufio.Reader) (string, error) {
	fmt.Print(prompt)
	output, err := reader.ReadString('\n')	
	output = strings.TrimSpace(output)
	return output, err
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	Config := ReadConfig()

	projectType, _ := input("What type of project is it? ", reader)
	projectStructure, exists := Config[projectType]
	if !exists {
		fmt.Println("Project Template Not Found")
		return
	}

	name, _ := input("What is the name of this project: ", reader)
	Project := NewProject(name, &projectStructure)

	// Creating the project diretory
	// TODO: Run BeforeHook
	err := os.Mkdir(Project.Path, 0750)
	if err != nil {
		fmt.Println("Error creating project directory")
		return
	}
}
