package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brainicorn/jsonschemagen/cmd"
	"github.com/brainicorn/jsonschemagen/schema"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	suite.Suite
}

// The entry point into the tests
func TestMainSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(MainTestSuite))
}

func (suite *MainTestSuite) SetupSuite() {
	command("go", "get", "-d", "-u", "-f", "github.com/brainicorn/schematestobjects").Run()
}

func command(name string, args ...interface{}) *exec.Cmd {
	var a []string
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			a = append(a, v)
		case []string:
			a = append(a, v...)
		}
	}
	c := exec.Command(name, a...)
	c.Stderr = os.Stderr
	return c
}

//func (suite *MainTestSuite) TestMainFunc() {
//	tmpDir := suite.getTempDir()
//	defer os.RemoveAll(tmpDir)

//	fmt.Println(len(os.Args))

//	if len(os.Args) > 1 {
//		os.Args[1] = "github.com/brainicorn/schematestobjects/album"
//	} else {
//		os.Args = append(os.Args, "github.com/brainicorn/schematestobjects/album")
//	}

//	os.Args = append(os.Args, "Album")
//	os.Args = append(os.Args, "-d")
//	os.Args = append(os.Args, "-o")
//	os.Args = append(os.Args, tmpDir)

//	defer clearArgs()
//	main()

//	assert.True(suite.T(), exists(filepath.Join(tmpDir, "github_com-brainicorn-schematestobjects-album-Album.json")))
//}

func (suite *MainTestSuite) TestBasicRun() {
	suite.T().Parallel()

	c := cmd.NewRootCommand()

	tmpDir := suite.getTempDir()
	defer os.RemoveAll(tmpDir)

	c.Cmd.SetArgs([]string{"-r", "-o", tmpDir, "github.com/brainicorn/schematestobjects/album", "Album"})
	err := c.Cmd.Execute()

	schemaFile := filepath.Join(tmpDir, pkgTypeToFilename("github.com/brainicorn/schematestobjects/album", "Album"))

	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists(schemaFile))

	s, err := readFileToSchema(schemaFile)

	assert.NoError(suite.T(), err)
	assert.True(suite.T(), len(s.GetDefinitions()) > 1, "empty definitions!")

}

func (suite *MainTestSuite) TestInlineDef() {
	suite.T().Parallel()

	c := cmd.NewRootCommand()

	tmpDir := suite.getTempDir()
	defer os.RemoveAll(tmpDir)

	c.Cmd.SetArgs([]string{"-q", "-o", tmpDir, "-i", "github.com/brainicorn/schematestobjects/album", "Album"})
	err := c.Cmd.Execute()

	schemaFile := filepath.Join(tmpDir, pkgTypeToFilename("github.com/brainicorn/schematestobjects/album", "Album"))

	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists(schemaFile))

	s, err := readFileToSchema(schemaFile)

	assert.NoError(suite.T(), err)

	assert.True(suite.T(), len(s.GetDefinitions()) < 1, "should not have definitions")

}

func (suite *MainTestSuite) TestDefFiles() {
	suite.T().Parallel()

	c := cmd.NewRootCommand()

	tmpDir := suite.getTempDir()
	defer os.RemoveAll(tmpDir)

	c.Cmd.SetArgs([]string{"-q", "-s", "-o", tmpDir, "github.com/brainicorn/schematestobjects/album", "Album"})
	err := c.Cmd.Execute()

	schemaFile := filepath.Join(tmpDir, pkgTypeToFilename("github.com/brainicorn/schematestobjects/album", "Album"))

	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists(schemaFile))

	s, err := readFileToSchema(schemaFile)

	assert.NoError(suite.T(), err)

	assert.True(suite.T(), len(s.GetDefinitions()) > 1, "should have definitions")
	assert.True(suite.T(), numFiles(tmpDir) > 2, "should have multiple definition files.")

}

func (suite *MainTestSuite) TestCodegen() {
	suite.T().Parallel()

	c := cmd.NewRootCommand()

	tmpDir := suite.getTempDir()
	defer os.RemoveAll(tmpDir)

	c.Cmd.SetArgs([]string{"-q", "-c", "-s", "-o", tmpDir, "github.com/brainicorn/schematestobjects/album", "Album"})
	err := c.Cmd.Execute()

	schemaFile := filepath.Join(tmpDir, pkgTypeToFilename("github.com/brainicorn/schematestobjects/album", "Album"))
	goFile := filepath.Join(tmpDir, "schema_accessor.go")

	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists(schemaFile), "schema file doesn't exist")
	assert.True(suite.T(), numFiles(tmpDir) > 2, "should have multiple definition files.")
	assert.True(suite.T(), exists(goFile), "golang file doesn't exist")

}

func (suite *MainTestSuite) TestUnmarshal() {
	suite.T().Parallel()

	c := cmd.NewRootCommand()

	tmpDir := suite.getTempDir()
	defer os.RemoveAll(tmpDir)

	c.Cmd.SetArgs([]string{"-q", "-s", "-o", tmpDir, "github.com/brainicorn/schematestobjects/album", "Album"})
	err := c.Cmd.Execute()

	schemaFile := filepath.Join(tmpDir, pkgTypeToFilename("github.com/brainicorn/schematestobjects/album", "Album"))

	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists(schemaFile), "schema file doesn't exist")

	schemaBytes, err := ioutil.ReadFile(schemaFile)

	assert.NoError(suite.T(), err)

	schemaObj, err := schema.FromJSON(schemaBytes)

	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), schema.SpecVersionDraftV4, schemaObj.GetSchemaURI(), "wrong uri")
	assert.Equal(suite.T(), "http://github.com/schemas/album", schemaObj.GetID(), "wrong id")
	assert.Equal(suite.T(), "An Album.", schemaObj.GetTitle(), "wrong title")
	assert.Equal(suite.T(), "A thing we used to play with a player and now we get from the air", schemaObj.GetDescription(), "wrong desc")

}

func clearArgs() {
	os.Args = []string{}
}

func (suite *MainTestSuite) getTempDir() string {
	dir, err := ioutil.TempDir("", "example")

	if err != nil {
		suite.Error(err)
	}

	return dir
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}

	return false
}

func numFiles(path string) int {
	files, err := ioutil.ReadDir(path)

	if err == nil {
		return len(files)
	}

	return 0
}

func readFileToSchema(path string) (schema.JSONSchema, error) {
	var err error
	var fbytes []byte

	fbytes, err = ioutil.ReadFile(path)

	if err == nil {
		s, err := schema.FromJSON(fbytes)
		return s, err
	}

	return nil, nil
}

func pkgTypeToFilename(p, t string) string {
	s := p + "/" + t
	s = strings.Replace(s, ".", "_", -1)
	s = strings.Replace(s, "/", "-", -1)

	s = s + ".json"

	return s

}
