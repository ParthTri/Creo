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

// TODO: Setup project structure inheritance
type TemplateStructure struct {
	Inherit									bool			`json:"inherit"`
	ProjectsDir							string		`json:"projectsDir"`
	ParentTemplate					string		`json:"parent"`
	parent									*TemplateStructure
	Git											bool			`json:"git"`	
	Gitignore								string		`json:"gitignore"`
	Env											bool			`json:"env"`
	IgnoreProjectPrexists		bool			`json:"ignoreProjectPrexists"`
	ExternalProgramsEnd			[]string	`json:"externalProgramsEnd"`
	ExternalProgramsStart		[]string	`json:"externalProgramsStart"`
	Dirs										[]string	`json:"dirs"`
	Files										[]string	`json:"files"`
}

type Template map[string]TemplateStructure

func (template *TemplateStructure)LookupParent (templates Template) error {
	for key, value := range templates {
		if key == template.ParentTemplate {
			template.parent = &value
			break
		}
	}

	if template.parent == nil {
		var errorString string = fmt.Sprintf("Parent template '%v' not found", template.ParentTemplate)

		return errors.New(errorString)
	}

	return nil
}

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
		fmt.Println(json)
		os.Exit(1)
	}

	return Project
}

type Project struct {
	Name					string							`arg:"$name" cli:"-n"`
	ProjectsDir		string							`arg:"$projectsDir"`
	Path					string							`arg:"$path"`
	TemplateName	string							`cli:"-t"`
	Structure			*TemplateStructure	
}

func (project *Project)GenerateProjectPaths(){
	if project.Structure.ProjectsDir != "" {
		project.ProjectsDir = project.Structure.ProjectsDir
		if string(project.ProjectsDir[0]) == "~" {
			projectsPath := strings.Split(project.ProjectsDir, "/")
			projectsPath[0], _ = os.UserHomeDir()
			project.ProjectsDir = strings.Join(projectsPath, "/")
		}
	} else {
		project.ProjectsDir, _ = os.UserHomeDir() 
		project.ProjectsDir += "/Projects/"
	}
	project.Path += project.ProjectsDir + project.Name
}

func (project *Project)loadArgs (args [][]string) {
	tagObject := reflect.TypeOf(*project)
	object := reflect.ValueOf(project).Elem()

	for i := 0; i < object.NumField(); i++ {
		field := object.Field(i)
		tagField := tagObject.Field(i)
		for _, arg := range args {
			tag := arg[0]
			if tag == string(tagField.Tag.Get("cli")) {
				field.SetString(arg[1])
			}
		}
	}
}

// Command line arguments
// --help
//		-t, --template		The template name
//		-n, --name				The name of the project
func GetCliArgs() ([][]string, error) {
	var Err error
	argsArray := os.Args[1:]

	if len(argsArray) % 2 != 0 {
		return [][]string{}, errors.New("Wrong number of arguments. Need at least 2.")
	} else if len(argsArray) > 4 {
		return [][]string{}, errors.New("Too many arguments. Maximum is 4.")
	}	

	test := Project{}
	object	:= reflect.TypeOf(test)
	identifiers := []int{}

	for k := 0; k < object.NumField() ; k++ {
		field := object.Field(k)
		tag := field.Tag.Get("cli")
		for i := 0; i < len(argsArray); i++ {
			if tag == argsArray[i] { identifiers = append(identifiers, i) }
		}
	}

	var args [][]string

	for i, j := 0, len(identifiers)-1; i < j; i, j = i+1, j-1 {
		identifiers[i], identifiers[j] = identifiers[j], identifiers[i]
	}

	for index, identifier := range identifiers {
		if index + 1 != len(identifiers) {
			elements := argsArray[identifier:identifiers[index+1]]
			args = append(args, elements)
		} else {
			elements := argsArray[identifier:]
			args = append(args, elements)
		}
	}

	return args, Err
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

// TODO: Rewrite function to use reflect.Value
func (project *Project)GetFieldValue(field string) string {
	tagObject := reflect.TypeOf(*project)
	object := reflect.ValueOf(project).Elem()

	for i := 0; i < object.NumField(); i++ {
		fieldVal := object.Field(i)
		tagField := tagObject.Field(i)
		if field == tagField.Tag.Get("arg") {
			return fieldVal.String()
		}
	}

	return ""
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

// Iterate over list of external programs listed in TemplateStructure and run them depending on the directory function provided
func (project Project)Hook(directory func(), point bool) error {
	var Err error
	var Range []string
	if point {
		Range = project.Structure.ExternalProgramsStart
	} else {
		Range = project.Structure.ExternalProgramsEnd
	}
	object := reflect.TypeOf(project)

	var interpolateData string
	var interpolateDataIndex int
	for _, command := range Range {
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

		directory()
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
	var projectStructure TemplateStructure
	var TargetProject *Project = &Project{}
	var Templates = make([]*Project, 2)
	Config := ReadConfig()
	args, err := GetCliArgs()

	if err == nil && len(args) > 0 {
		for _, arg := range args {
			if arg[0] == "-t" {
				projectStructure = Config[args[0][1]]
				TargetProject.Structure = &projectStructure
			}
		}
	}
	if len(args) > 0 {
		TargetProject.loadArgs(args)	
	}

	reader := bufio.NewReader(os.Stdin)
	if TargetProject.Structure == nil {
		projectType, _ := input("What type of project is it? ", reader)
		projectStructure, exists := Config[projectType]

		if !exists {
			fmt.Println("TargetProject Template Not Found")
			return
		}
		TargetProject.Structure = &projectStructure
		TargetProject.TemplateName = projectType
	}
	
	if TargetProject.Name == "" {
		name, _ := input("What is the name of this project: ", reader)
		TargetProject.Name = name
	}

	if TargetProject.Structure.Inherit {
		err = TargetProject.Structure.LookupParent(Config)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		} 

		parent := *TargetProject
		parent.Structure = TargetProject.Structure.parent
		Templates[0] = &parent
		Templates[1] = TargetProject
	} else {
		Templates[0] = TargetProject
	}

	// BUG: Project inheritence can have conflicts when creating .env and .gitignore files
	// TODO: Need to make sure child templates overwrite these configurations

	for _, project := range Templates{
		if project == nil {
			return
		}

		project.GenerateProjectPaths()
		// Run BeforeHook
		if len(project.Structure.ExternalProgramsStart) != 0 {
			fmt.Println(project.Hook(func () {os.Chdir(project.ProjectsDir)}, true))
		}

		// Creating the project diretory
		err = os.Mkdir(project.Path, 0750)
		if os.IsExist(err) && project.Structure.IgnoreProjectPrexists { }	else if err != nil {
			fmt.Println("Error creating project directory")
			fmt.Println(err)
			return
		}

		if project.Structure.Git {
			status := project.Git()
			if status != nil {
				fmt.Println("Error with git")
				fmt.Println(status)
				return
			}
		}

		if project.Structure.Env {
			path := fmt.Sprintf("%v/.env", project.Path)
			_, err := os.Create(path)
			if err != nil {
				fmt.Println("Error creating .env file")
			}
		}

		if len(project.Structure.Dirs) != 0 {
			err := project.CreateDirectories()
			if err != nil {
				fmt.Println("Error Creating Sub-Directories")
				fmt.Println(err)
				return 
			}
		}

		if len(project.Structure.Files) != 0 {
			err := project.CreateFiles()
			if err != nil {
				fmt.Println("Error Creating Files")
				fmt.Println(err)
				return 
			}
		}

		if len(project.Structure.ExternalProgramsEnd) != 0 {
			err := project.Hook(func () {os.Chdir(project.Path)}, false)
			if err != nil {
				fmt.Println("Error running")
				fmt.Println(err)
				return
			}
		}
	}
}
