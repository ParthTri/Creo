package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
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
	Name				string							`arg:"$name"`
	ProjectsDir	string							`arg:"$projectsDir"`
	Path				string							`arg:"$path"`
	Structure		*TemplateStructure
}

// TODO: Allow for user to set the path to their Projects directory or use the one supplied by the project structure
func NewProject(name string, structure *TemplateStructure) Project {
	project := Project{
		Name:				name,
		Structure: structure,
	}

	project.ProjectsDir, _ = os.UserHomeDir() 
	project.ProjectsDir += "/Projects/"
	project.Path += project.ProjectsDir + project.Name

	return project
}

// Function to iterate over directories in TemplateStructure.Dirs and create them 
// Allows for implicit directories i.e. "src/templates"
func (project Project)CreateDirectories() error {
	var Err error
	for _, directory := range project.Structure.Dirs {
		path := fmt.Sprintf("%v/%v", project.Path, directory)
		err := os.MkdirAll(path, 0750)
		if err != nil {
			Err = err	
		}
	}	
	return Err
}

// Function to iterate over files in TemplateStructure.Files and create them
// Make sure that the directory already exists for verboseness
func (project Project)CreateFiles() error{
	var Err error
	for _, file := range project.Structure.Files {
		path := fmt.Sprintf("%v/%v", project.Path, file)
		_, err := os.Create(path)
		if err != nil {
			Err = err
		}
	}
	return Err
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

// Iterate over external programs as listed in TemplateStructure.ExternalProgramsEnd and run them at the end of project construction
func (project Project)AfterHook() error {
	var Err error
	for _, command := range project.Structure.ExternalProgramsEnd {
		commandSplit := strings.Split(command, " ")	
		os.Chdir(project.Path)

		cmd := exec.Command(commandSplit[0], commandSplit[1:]...)
		err := cmd.Run()
		if err != nil {
			Err = err
		}
	}
	return Err
}

func (project Project)GetFieldValue(field string) string {
	fieldsMap := map[string]string{
		"$name": project.Name,
		"$projectsDir": project.ProjectsDir,
		"$path": project.Path,
	}
	return fieldsMap[field]
}

func (project Project)GetInterpolateData(object reflect.Type, arg string) (string, error) {
	for i := 0; i < object.NumField(); i++ {
		field := object.Field(i)
		if tag := field.Tag.Get("arg"); tag == arg {
			return project.GetFieldValue(tag), nil
		}
	}
	return "", errors.New("Tag Not Found")
}

// Iterate over external programs as listed in TemplateStructure.ExternalProgramsStart and run them at the start of project construction before the project directory is made 
// BUG: change directory (cd) doesn't work. Need to change current directory through os.Chdir()
func (project Project)BeforeHook() error {
	var Err error
	object := reflect.TypeOf(project)

	var interpolateData string
	var interpolateDataIndex int
	for _, command := range project.Structure.ExternalProgramsStart {
		commandSplit := strings.Split(command, " ")		
		for index, arg := range commandSplit {
			if string(arg[0]) == "$" {
				data, err := project.GetInterpolateData(object, arg)
				if err != nil {
					return err
				}
				interpolateData = data	
				interpolateDataIndex = index
				break
			}
		}
		if interpolateData != "" {
			commandSplit[interpolateDataIndex] = interpolateData
		}

		os.Chdir(project.ProjectsDir)
		cmd := exec.Command(commandSplit[0], commandSplit[1:]...)
		err := cmd.Run()
		if err != nil {
			Err = err
		}
	}
	return Err
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
	
	if Project.Structure.Git {
		status := Project.Git()
		if status != nil {
			fmt.Println("Error with git")
			fmt.Println(status)
			return
		}
	}
	
	if Project.Structure.Env {
		path := fmt.Sprintf("%v/.env", Project.Path)
		_, err := os.Create(path)
		if err != nil {
			fmt.Println("Error creating .env file")
		}
	}
	
	if len(Project.Structure.Dirs) != 0 {
		err := Project.CreateDirectories()
		if err != nil {
			fmt.Println("Error Creating Sub-Directories")
			fmt.Println(err)
			return 
		}
	}
	
	if len(Project.Structure.Files) != 0 {
		err := Project.CreateFiles()
		if err != nil {
			fmt.Println("Error Creating Files")
			fmt.Println(err)
			return 
		}
	}

	if len(Project.Structure.ExternalProgramsEnd) != 0 {
		err := Project.AfterHook()
		if err != nil {
			fmt.Println("Error running")
			fmt.Println(err)
			return
		}
	}
}
