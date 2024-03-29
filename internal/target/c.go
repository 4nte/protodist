package target

import (
	"fmt"
	"github.com/4nte/protodist/git"
	"github.com/4nte/protodist/util"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
)

func C(protoOutDir string, gitCfg git.Config, cloneBranch string, cloneDir string, dryRun bool, deployTarget string, deployDir string) {
	filterPackages := []string{"gateway", "device"}
	var scannedPackages []string

	// Scan compiled go packages
	files, err := ioutil.ReadDir(path.Join(protoOutDir, "c"))
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if !f.IsDir() {
			panic("file is not expected to be here, only directories")
		}
		scannedPackages = append(scannedPackages, f.Name())
		//fmt.Println(f.Path())
	}

	// Append packages to cPackages that match the filter
	var cPackages []string
	for _, pkg := range scannedPackages {
		for _, filterPkg := range filterPackages {
			if pkg == filterPkg {
				cPackages = append(cPackages, pkg)
				break
			}
		}
	}

	// Clone C proto repos
	for _, pkg := range cPackages {
		repoName := fmt.Sprintf("proto-%s-c", pkg)
		repoUrl := gitCfg.GetRepoURL(repoName)
		git.Clone(repoUrl, cloneBranch)
	}

	// Delete all .go files in cloned go proto repos
	for _, pkg := range cPackages {
		repoName := fmt.Sprintf("proto-%s-c", pkg)
		repoDir := path.Join(cloneDir, repoName)
		pkgCloneDir, err := ioutil.ReadDir(repoDir)
		if err != nil {
			panic(err)
		}

		// Search for files with .go extension && delete them
		for _, file := range pkgCloneDir {

			// Filter directories
			if file.IsDir() && file.Name() != pkg {
				continue
			}
			// skip deleting files which are not .c or .h
			if filepath.Ext(file.Name()) != ".c" && filepath.Ext(file.Name()) != ".h" {
				continue
			}
			// Delete file
			err := os.Remove(file.Name())
			if err != nil {
				panic(err)
			}
		}

		// Move generate .c files to cloned repo dir
		generatedPkgDirPath := path.Join(protoOutDir, "c", pkg)
		err = util.CopyDirectory(generatedPkgDirPath, repoDir)
		if err != nil {
			panic(err)
		}

		// nanopb imports fix
		// generated C files expect header files to reside in `package/foo.pb.h` path, but they are all in the same directory
		// Here we are creating a sub-directory with the name of a proto package, and moving the header files into it.
		headerDirPath := path.Join(generatedPkgDirPath, pkg)
		err = util.CreateIfNotExists(headerDirPath, 0755)
		if err != nil {
			panic(err)
		}

		generatedPkgDir, err := ioutil.ReadDir(generatedPkgDirPath)
		if err != nil {
			panic(err)
		}
		for _, file := range generatedPkgDir {
			if filepath.Ext(file.Name()) == ".h" {
				originalFile := path.Join(generatedPkgDirPath, file.Name())
				err = util.Copy(path.Join(originalFile), path.Join(headerDirPath, file.Name()))
				if err != nil {
					panic(err)
				}

				// Delete the original file
				err := os.Remove(originalFile)
				if err != nil {
					panic(err)
				}
			}
		}

	}

	var repoNames []string
	for _, cPkg := range cPackages {
		repoNames = append(repoNames, fmt.Sprintf("proto-%s-c", cPkg))
	}
	AddCommitTagPush(gitCfg, repoNames, dryRun)
}
