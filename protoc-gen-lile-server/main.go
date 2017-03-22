package main

import (
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/fatih/color"
	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

var outputPath string
var templatePath = os.Getenv("GOPATH") + "/src/github.com/lileio/lile/protoc-gen-lile-server/templates"

type grpcMethod struct {
	GoPackage string
	Name      string
	InType    string
	OutType   string
}

func main() {
	// Force color output
	color.NoColor = false

	// Parse the incoming protobuf request
	req, err := parseReq(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	path := *req.Parameter
	if path == "" {
		path = "./server"
	}

	for _, file := range req.ProtoFile {
		if file.Options == nil || file.Options.GoPackage == nil {
			log.Fatalf("No go_package option defined in %s", *file.Name)
		}

		for _, service := range file.Service {
			for _, method := range service.Method {
				gm := grpcMethod{
					GoPackage: *file.Options.GoPackage,
					Name:      *method.Name,
					InType:    strings.Trim(*method.InputType, "."),
					OutType:   strings.Trim(*method.OutputType, "."),
				}

				err := generateMethod(path, gm)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}

func generateMethod(basePath string, m grpcMethod) error {
	path := filepath.Join(basePath, strings.ToLower(m.Name)+".go")

	// If the file exists then just skip the creation
	_, err := os.Stat(path)
	if err == nil {
		log.Printf("%s %s", color.YellowString("[Skipping]"), path)
		return nil
	}

	log.Printf("%s %s", color.GreenString("[Creating]"), path)
	return render(path, m)
}

func render(path string, m grpcMethod) error {
	t, err := template.ParseFiles(filepath.Join(templatePath, "unary_unary.tmpl"))
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	err = t.Execute(file, m)
	if err != nil {
		return err
	}

	return nil
}

func parseReq(r io.Reader) (*plugin.CodeGeneratorRequest, error) {
	input, err := ioutil.ReadAll(r)
	if err != nil {
		log.Errorf("Failed to read code generator request: %v", err)
		return nil, err
	}

	req := new(plugin.CodeGeneratorRequest)
	if err = proto.Unmarshal(input, req); err != nil {
		log.Errorf("Failed to unmarshal code generator request: %v", err)
		return nil, err
	}

	return req, nil
}

func emitFiles(out []*plugin.CodeGeneratorResponse_File) {
	emitResp(&plugin.CodeGeneratorResponse{File: out})
}

func emitError(err error) {
	emitResp(&plugin.CodeGeneratorResponse{Error: proto.String(err.Error())})
}

func emitResp(resp *plugin.CodeGeneratorResponse) {
	buf, err := proto.Marshal(resp)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stdout.Write(buf); err != nil {
		log.Fatal(err)
	}
}
