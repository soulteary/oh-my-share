package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Project struct {
	Name      string    `json:"name"`
	FullName  string    `json:"full_name"`
	Private   bool      `json:"private"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	PushedAt  time.Time `json:"pushed_at"`
	License   struct {
		Key  string `json:"key"`
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"license"`
}

const CACHE_DIR = "cache"

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	maxPage := 4

	fetchWithCache(token, maxPage)
	projects := mergeProjectData()
	fmt.Println(len(projects))
}

func mergeProjectData() (allProjects []Project) {
	files, err := os.ReadDir(CACHE_DIR)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		buf, err := os.ReadFile(fmt.Sprintf("cache/%s", file.Name()))
		if err != nil {
			panic(err)
		}
		var projects []Project
		json.Unmarshal(buf, &projects)

		for pId := range projects {
			if !projects[pId].Private {
				allProjects = append(allProjects, projects[pId])
			}
		}
	}

	buf, err := json.Marshal(allProjects)
	if err != nil {
		panic(err)
	}
	os.WriteFile("projects.json", buf, os.ModePerm)

	return allProjects
}

func fetchWithCache(token string, maxPage int) {
	os.MkdirAll(CACHE_DIR, os.ModePerm)
	for i := 1; i <= maxPage; i++ {
		info, err := os.Stat(fmt.Sprintf("cache/%d.json", i))
		if err == nil && time.Now().Sub(info.ModTime()).Hours() < 24 {
			continue
		}
		buf, err := fetchData(i, token)
		if err != nil {
			panic(err)
		}
		os.WriteFile(fmt.Sprintf("cache/%d.json", i), buf, os.ModePerm)
	}
}

// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#list-repositories-for-a-user
func fetchData(page int, token string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/users/soulteary/repos?per_page=100&page=%d", page), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
