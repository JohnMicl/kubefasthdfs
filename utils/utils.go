package utils

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

func ParseYamlFile(filePath string, obj interface{}) error {
	byteData, err := ioutil.ReadFile(GetAbsDir(filePath))
	if err != nil {
		return err
	}
	return yaml.Unmarshal(byteData, obj)
}

func GetAbsDir(relativePath string) string {
	dir := filepath.Dir(os.Args[0])
	return path.Join(dir, relativePath)
}
