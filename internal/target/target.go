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

func Golang(protoOutDir string, gitCfg git.Config, cloneBranch string, cloneDir string) {
	var goPackages []string
	// Scan compiled go packages
	files, err := ioutil.ReadDir(path.Join(protoOutDir, "go"))
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if !f.IsDir() {
			panic("file is not expected to be here, only directories")
		}
		goPackages = append(goPackages, f.Name())
		fmt.Println(f.Name())
	}
	//defer os.RemoveAll(cloneDir)

	// Clone go proto repos
	for _, pkg := range goPackages {
		repoName := fmt.Sprintf("proto-%s-go", pkg)
		repoUrl := gitCfg.GetRepoURL(repoName)
		git.Clone(repoUrl, cloneBranch)
	}

	// Delete all .go files in cloned go proto repos
	for _, pkg := range goPackages {
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
			err := os.Remove(file.Name())
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

	}

	var repoNames []string
	for _, goPkg := range goPackages {
		repoNames = append(repoNames, fmt.Sprintf("proto-%s-go", goPkg))
	}
	AddCommitTagPush(gitCfg, repoNames)
}

func AddCommitTagPush(cfg git.Config, repos []string) {
	for _, repo := range repos {
		git.AddAll(repo)
		git.Commit(repo, "add pb files")

		refType, refName := cfg.ParseRef()
		if refType == git.BranchRef {
			git.Push(repo, refName)

		} else {
			git.Tag(repo, refName)
		}
	}
}

func Javascript(protoOutDir string, gitCfg git.Config, cloneBranch string, cloneDir string) {
	var tsPackages []string

	repoName := "proto-all-js"
	repoUrl := gitCfg.GetRepoURL(repoName)
	git.Clone(repoUrl, cloneBranch)

	packageDirs, err := ioutil.ReadDir(path.Join(protoOutDir, "ts"))
	if err != nil {
		log.Fatal(err)
	}
	for _, pkgDir := range packageDirs {
		if !pkgDir.IsDir() {
			panic("file is not expected to be here, only directories")
		}
		tsPackages = append(tsPackages, pkgDir.Name())
	}

	// Copy generated pb files to repo dirs
	for _, pkg := range tsPackages {
		pkgTargetDir := path.Join(cloneDir, repoName, pkg)
		if err := os.MkdirAll(pkgTargetDir, 0700); err != nil {
			panic(fmt.Errorf("failed to create dir for package: %s", err))
		}
		err := util.CopyDirectory(path.Join(protoOutDir, "ts", pkg), path.Join(cloneDir, repoName, pkg))
		if err != nil {
			panic(err)
		}

	}

	// Add to GIT
	AddCommitTagPush(gitCfg, []string{repoName})
}
