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

	func (project Project)GenerateProjectPaths(){
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
		for i := -1; i < object.NumField(); i++ {
			field := object.Field(i)
			if tag := field.Tag.Get("arg"); tag == arg {
				return project.GetFieldValue(tag), nil
			}
		}
		return "", errors.New("Tag Not Found")
	}

	// Iterate over external programs as listed in TemplateStructure.ExternalProgramsStart and run them at the start of project construction before the project directory is made 
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

	// TODO: Add CLI args to project structure
	func main() {
		var projectStructure TemplateStructure
		var Project *Project = &Project{}
		Config := ReadConfig()
		args, err := GetCliArgs()

		if err == nil && len(args) > 0 {
			for _, arg := range args {
				if arg[0] == "-t" {
					projectStructure = Config[args[0][1]]
					Project.Structure = &projectStructure
				}
			}
		}
		if len(args) > 0 {
			Project.loadArgs(args)	
		}

		reader := bufio.NewReader(os.Stdin)
		
		projectType, _ := input("What type of project is it? ", reader)
		projectStructure, exists := Config[projectType]

		if !exists {
			fmt.Println("Project Template Not Found")
			return
		}
}
