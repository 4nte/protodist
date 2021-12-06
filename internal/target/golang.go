package target

import (
	"bytes"
	"fmt"
	"github.com/4nte/protodist/git"
	"github.com/4nte/protodist/util"
	"github.com/pkg/errors"
	"go/parser"
	"go/token"
	"golang.org/x/tools/go/packages"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

const GoModTemplate = `
module {{ .ModulePath }}

go {{ .GoVersion }}

{{ range .RequiredPackages -}}
{{ if .LocalPath }}
replace {{ .Path }} => {{ .LocalPath }}
{{ end }}
{{- end -}}

require (
	{{ range .RequiredPackages -}}
	{{ .Path }} {{ .Version }}
	{{ end -}}
)
`

var standardPackages = make(map[string]struct{})

func loadStandardPackages() {
	pkgs, err := packages.Load(nil, "std")
	if err != nil {
		panic(err)
	}

	for _, p := range pkgs {
		standardPackages[p.PkgPath] = struct{}{}
	}
	isStandardPackage("sync")
	//fmt.Println("pkgs", pkgs)
	//fmt.Println("is std", isStandardPackage("sync"))
}
func isStandardPackage(pkg string) bool {
	_, ok := standardPackages[pkg]
	return ok
}

// Go Module
type Module struct {
	Path      string
	Version   string
	LocalPath string
}

var knownPackages = []Module{
	{Path: "github.com/golang/protobuf", Version: "v1.4.3"},
	{Path: "google.golang.org/protobuf", Version: "v1.25.0"},
	{Path: "google.golang.org/grpc", Version: "v1.35.0"},
}

func parseImports(filename string) []string {
	var imports []string
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ImportsOnly)
	if err != nil {
		log.Fatal(err)
	}
	for _, imp := range node.Imports {
		imports = append(imports, strings.ReplaceAll(imp.Path.Value, `"`, ""))
	}

	return imports
}

type ModuleResolverFunc func(module string, requiredPackages []Module) string

type DependencyResolver struct {
	modules map[string]*struct {
		//Commit *git.CommitInfo
		Version                 *string
		RequiredProtoFamilyDeps []string
		RequiredThirdPartyDeps  []Module
		LocalPath               string
	}

	// Implement `git commit` for module
	resolverFunc ModuleResolverFunc
}

func NewDependencyResolver(ResolverFunc ModuleResolverFunc) DependencyResolver {
	return DependencyResolver{
		resolverFunc: ResolverFunc,
		modules: make(map[string]*struct {
			Version                 *string
			RequiredProtoFamilyDeps []string
			RequiredThirdPartyDeps  []Module
			LocalPath               string
		}),
	}
}
func (r DependencyResolver) AddModule(module string, localPath string, requiredDeps []string, requiredThirdPartyDeps []Module) {
	r.modules[module] = &struct {
		Version                 *string
		RequiredProtoFamilyDeps []string
		RequiredThirdPartyDeps  []Module
		LocalPath               string
	}{Version: nil, LocalPath: localPath, RequiredProtoFamilyDeps: requiredDeps, RequiredThirdPartyDeps: requiredThirdPartyDeps}
}

func (r DependencyResolver) resolveModule(module string, version string) {
	r.modules[module].Version = &version
}
func (r DependencyResolver) isModuleResolved(module string) bool {
	return r.modules[module].Version != nil
}

func (r DependencyResolver) Resolve() {
	for modulePath, module := range r.modules {
		// Skip module if already resolved
		if r.isModuleResolved(modulePath) {
			continue
		}

		areAllDepsResolved := true
		for _, requiredDep := range module.RequiredProtoFamilyDeps {
			if !r.isModuleResolved(requiredDep) {
				areAllDepsResolved = false
				break
			}
		}

		// All deps must be resolved before resolving the module itself
		if areAllDepsResolved {
			var requiredDeps []Module
			// Add proto family deps
			for _, dep := range module.RequiredProtoFamilyDeps {
				depModule := r.modules[dep]
				requiredDeps = append(requiredDeps, Module{
					Path:      dep,
					Version:   *depModule.Version,
					LocalPath: depModule.LocalPath,
				})
			}
			// Add third party deps
			requiredDeps = append(requiredDeps, module.RequiredThirdPartyDeps...)
			ver := r.resolverFunc(modulePath, requiredDeps)
			module.Version = &ver
		}

	}

	// If there are modules left unresolved, Resolve() them.
	for modulePath, _ := range r.modules {
		if !r.isModuleResolved(modulePath) {
			r.Resolve()
		}
	}
}
func Golang(protoOutDir string, gitCfg git.Config, cloneBranch string, cloneDir string, dryRun bool, deployTarget string, deployDir string) {
	var protoModules []string // Currently compiled proto modules
	loadStandardPackages()
	var goPackages []string
	// Scan compiled go packages
	files, err := ioutil.ReadDir(path.Join(protoOutDir, "go"))
	if err != nil {
		log.Fatal(err)
	}

	//_, refName := gitCfg.ParseRef()
	for _, f := range files {
		if !f.IsDir() {
			panic("file is not expected to be here, only directories")
		}
		pkgName := f.Name()
		goPackages = append(goPackages, pkgName)

		// Add proto module
		modulePath := fmt.Sprintf("%s/proto-%s-go", gitCfg.GitBase(), pkgName)
		protoModules = append(protoModules, modulePath)

		//fmt.Println(f.Path())
	}
	//defer os.RemoveAll(cloneDir)

	// Clone go proto repos
	for _, pkg := range goPackages {
		repoName := fmt.Sprintf("proto-%s-go", pkg)
		if deployTarget == "git" {
			repoUrl := gitCfg.GetRepoURL(repoName)
			git.Clone(repoUrl, cloneBranch)
		} else if deployTarget == "local" {
			err := os.Mkdir(path.Join(os.TempDir(), repoName), 0755)
			if err != nil {
				panic(errors.Wrap(err, "failed to create a dir"))
			}
		}

	}

	depResolver := NewDependencyResolver(func(modulePath string, requiredPackages []Module) string {
		fmt.Println("resolving module", modulePath)

		type GoModData struct {
			ModulePath       string
			GoVersion        string
			RequiredPackages []Module
		}

		data := GoModData{
			ModulePath:       modulePath,
			GoVersion:        "1.15",
			RequiredPackages: requiredPackages,
		}

		tmpl, err := template.New("gomod").Parse(GoModTemplate)
		buffer := bytes.NewBuffer(nil)
		if err := tmpl.Execute(buffer, data); err != nil {
			panic(err)
		}

		fragments := strings.SplitAfter(modulePath, "/")
		repoName := fragments[len(fragments)-1]
		f, err := os.OpenFile(path.Join(cloneDir, repoName, "go.mod"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			log.Fatal(err)
		}

		_, err = f.Write(buffer.Bytes())
		if err != nil {
			panic(err)
		}

		if err := f.Close(); err != nil {
			log.Fatal(err)
		}

		var moduleVersion string

		if deployTarget == "git" {
			git.AddAll(repoName)
			commit := git.Commit(repoName, "add pb files")
			refType, refName := gitCfg.ParseRef()
			if refType == git.TagRef {
				// Create a git tag
				git.Tag(repoName, refName)
			}
			if !dryRun {
				git.Push(repoName, refName)
			}

			switch refType {
			case git.BranchRef:
				moduleVersion = commit.ModulePseudoVersion()
			case git.TagRef:
				moduleVersion = refName
			}
		} else if deployTarget == "local" {
			repoDir := path.Join(os.TempDir(), repoName)
			// Create module dir
			moduleDir := fmt.Sprintf("%s/%s", deployDir, repoName)
			err := os.Mkdir(moduleDir, 0755)
			if err != nil {
				panic(err)
			}
			// Copy contents into module dir
			err = util.CopyDirectory(repoDir, moduleDir)
			if err != nil {
				panic(errors.Wrap(err, "failed to copy repo dir to deploy dir "))
			}

			moduleVersion = "v0.0.0-local"
		}

		return moduleVersion
	})

	for _, pkg := range goPackages {
		modulePath := fmt.Sprintf("%s/proto-%s-go", gitCfg.GitBase(), pkg)
		repoName := fmt.Sprintf("proto-%s-go", pkg)
		repoDir := path.Join(cloneDir, repoName)
		pkgCloneDir, err := ioutil.ReadDir(repoDir)
		if err != nil {
			panic(err)
		}

		// Search for files with .go extension && delete them
		for _, file := range pkgCloneDir {
			if !file.Mode().IsRegular() || filepath.Ext(file.Name()) != ".go" {
				continue
			}
			// Delete .go file
			err := os.Remove(path.Join(repoDir, file.Name()))
			if err != nil {
				panic(err)
			}
		}

		// Move generate .go files to cloned repo dir
		generatedPkgDir := path.Join(protoOutDir, "go", pkg)
		err = util.CopyDirectory(generatedPkgDir, repoDir)
		if err != nil {
			panic(err)
		}

		// Generate go.mod file for Module
		var importedPackages []string
		entries, err := ioutil.ReadDir(generatedPkgDir)
		for _, file := range entries {
			if file.IsDir() {
				continue
			}
			if filepath.Ext(file.Name()) != ".go" {
				continue
			}

			// Imported packages discovery
			imports := parseImports(path.Join(generatedPkgDir, file.Name()))
			for _, importedPkg1 := range imports {
				// Ignore packages from standard library
				if isStandardPackage(importedPkg1) {
					continue
				}
				var isFound bool
				for _, importedPkg2 := range importedPackages {
					if importedPkg1 == importedPkg2 {
						isFound = true
						continue
					}
				}
				if !isFound {
					importedPackages = append(importedPackages, importedPkg1)
				}
			}
		}

		// Resolve packages
		var requiredThirdPartyPackages []Module
		var requiredProtoPackages []string // These must be a string because they aren't resolved (just Path, no version known until a git commit is made)
		var unknownPackages []string
		for _, importedPkg := range importedPackages {
			if isStandardPackage(importedPkg) {
				continue
			}

			var isFound bool
			// Test if pkg is a family member proto module
			for _, module := range protoModules {
				if strings.HasPrefix(importedPkg, module) {
					isFound = true
					// Check if pkg was already added to required proto packages
					var isAlreadyAdded bool
					for _, addedPkg := range requiredProtoPackages {
						if addedPkg == importedPkg {
							isAlreadyAdded = true
						}
					}

					if isAlreadyAdded {
						continue
					}

					requiredProtoPackages = append(requiredProtoPackages, module)
					continue
				}
			}
			// Test if pkg is in known 3d party packages
			for _, knownPkg := range knownPackages {
				// Check if import is a known package
				if strings.HasPrefix(importedPkg, knownPkg.Path) {
					isFound = true
					// Check if pkg was already added to requiredThirdPartyPackages
					var isAlreadyAdded bool
					for _, addedPkg := range requiredThirdPartyPackages {
						if addedPkg.Path == knownPkg.Path {
							isAlreadyAdded = true
						}
					}

					if isAlreadyAdded {
						continue
					}

					requiredThirdPartyPackages = append(requiredThirdPartyPackages, knownPkg)
					continue
				}
			}

			if !isFound {
				// Package not identified, I don't know what to do with it.
				unknownPackages = append(unknownPackages, importedPkg)
			}

		}

		if len(unknownPackages) > 0 {
			for _, unresolvedPackage := range unknownPackages {
				fmt.Printf("failed to resolve package %s\n", unresolvedPackage)
			}
			panic(fmt.Sprintf("failed to resolve %d packages\n", len(unknownPackages)))
		}

		var localPath string
		if deployTarget == "local" {
			localPath = path.Join("../", repoName)
		}
		depResolver.AddModule(modulePath, localPath, requiredProtoPackages, requiredThirdPartyPackages)
	}

	// Resolve all deps
	depResolver.Resolve()

}
