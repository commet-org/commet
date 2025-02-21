package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const version = "0.1.0"

type Commit struct {
	Hash      string   `json:"hash"`
	Message   string   `json:"message"`
	Timestamp string   `json:"timestamp"`
	Files     []string `json:"files"`
}

type Repo struct {
	RepoDir  string
	VcsDir   string
}

func NewRepo(repoDir string) *Repo {
	vcsDir := filepath.Join(repoDir, ".commet")
	return &Repo{RepoDir: repoDir, VcsDir: vcsDir}
}

func (r *Repo) Init() error {
	if _, err := os.Stat(r.VcsDir); !os.IsNotExist(err) {
		return fmt.Errorf("repository already initialized")
	}
	if err := os.Mkdir(r.VcsDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to initialize repository: %v", err)
	}
	fmt.Println("Initialized empty repository in", r.RepoDir)
	return nil
}

func (r *Repo) HashFile(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha1.New()
	if _, err := file.WriteTo(hasher); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (r *Repo) Add(filePath string) error {
	stagedFile := filepath.Join(r.VcsDir, "staged.json")
	fileHash, err := r.HashFile(filePath)
	if err != nil {
		return err
	}
	fileData := map[string]string{
		"path": filePath,
		"hash": fileHash,
	}
	var staged []map[string]string
	if _, err := os.Stat(stagedFile); err == nil {
		file, err := os.Open(stagedFile)
		if err != nil {
			return err
		}
		defer file.Close()
		if err := json.NewDecoder(file).Decode(&staged); err != nil {
			return err
		}
	}
	staged = append(staged, fileData)
	file, err := os.Create(stagedFile)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(staged); err != nil {
		return err
	}
	fmt.Printf("Added %s to staging area\n", filePath)
	return nil
}

func (r *Repo) Commit(message string) error {
	stagedFile := filepath.Join(r.VcsDir, "staged.json")
	file, err := os.Open(stagedFile)
	if err != nil {
		return fmt.Errorf("no changes to commit")
	}
	defer file.Close()
	var staged []map[string]string
	if err := json.NewDecoder(file).Decode(&staged); err != nil {
		return fmt.Errorf("failed to read staged files: %v", err)
	}
	commitHash := sha1.New()
	commitHash.Write([]byte(message + time.Now().String()))
	hash := hex.EncodeToString(commitHash.Sum(nil))
	commit := Commit{
		Hash:      hash,
		Message:   message,
		Timestamp: time.Now().String(),
		Files:     []string{},
	}
	commitDir := filepath.Join(r.VcsDir, "commits")
	if err := os.MkdirAll(commitDir, os.ModePerm); err != nil {
		return err
	}
	commitFile := filepath.Join(commitDir, commit.Hash)
	commitData, err := json.Marshal(commit)
	if err != nil {
		return err
	}
	if err := os.WriteFile(commitFile, commitData, os.ModePerm); err != nil {
		return err
	}
	os.Remove(stagedFile)
	fmt.Println("Commit successful:", message)
	return nil
}

func (r *Repo) Status() error {
	stagedFile := filepath.Join(r.VcsDir, "staged.json")
	if _, err := os.Stat(stagedFile); os.IsNotExist(err) {
		fmt.Println("No changes staged.")
		return nil
	}
	file, err := os.Open(stagedFile)
	if err != nil {
		return err
	}
	defer file.Close()
	var staged []map[string]string
	if err := json.NewDecoder(file).Decode(&staged); err != nil {
		return err
	}
	if len(staged) == 0 {
		fmt.Println("No changes staged.")
	} else {
		fmt.Println("Changes staged:")
		for _, file := range staged {
			fmt.Printf("- %s\n", file["path"])
		}
	}
	return nil
}

func printHelp() {
	fmt.Println("Commet - A simple Git-like tool written in Go")
	fmt.Println("\nUsage:")
	fmt.Println("  commet [command] [options]\n")
	fmt.Println("Available commands:")
	fmt.Println("  init      Initialize a new repository")
	fmt.Println("  add       Stage a file")
	fmt.Println("  commit    Commit staged changes")
	fmt.Println("  status    Show the status of the repository")
	fmt.Println("  -v        Show version information")
	fmt.Println("\nUse 'commet [command] -h' for more information about a command.")
}

func main() {
	versionFlag := flag.Bool("v", false, "Show version information")
	helpFlag := flag.Bool("help", false, "Show help")
	flag.Parse()

	if *versionFlag {
		fmt.Println("Commet version:", version)
		return
	}

	if *helpFlag || flag.NArg() == 0 {
		printHelp()
		return
	}

	repo := NewRepo("./")
	switch flag.Arg(0) {
	case "init":
		err := repo.Init()
		if err != nil {
			fmt.Println(err)
		}
	case "add":
		if flag.NArg() < 2 {
			fmt.Println("Error: You must specify a file to add.")
			return
		}
		err := repo.Add(flag.Arg(1))
		if err != nil {
			fmt.Println(err)
		}
	case "commit":
		if flag.NArg() < 2 {
			fmt.Println("Error: You must provide a commit message.")
			return
		}
		err := repo.Commit(flag.Arg(1))
		if err != nil {
			fmt.Println(err)
		}
	case "status":
		err := repo.Status()
		if err != nil {
			fmt.Println(err)
		}
	default:
		printHelp()
	}
}
