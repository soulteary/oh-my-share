package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"time"
)

type Project struct {
	Name        string    `json:"name"`
	FullName    string    `json:"full_name"`
	Description string    `json:"description"`
	URL         string    `json:"html_url"`
	Homepage    string    `json:"homepage"`
	Private     bool      `json:"private"`
	Fork        bool      `json:"fork"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	PushedAt    time.Time `json:"pushed_at"`
	License     struct {
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
	makeTemplate(projects)
}

func makeTemplate(projects []Project) {
	baseTemplate, err := os.ReadFile("template/index.html")
	if err != nil {
		panic(err)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[j].PushedAt.Before(projects[i].PushedAt)
	})

	var template = ""
	for _, project := range projects {
		template += `
		<figure class="project">
			<div class="preview">
				<img src="projects/` + strings.ToLower(project.FullName) + `" alt="" onerror="this.src='placeholder.jpg'" />
			</div>
			<div class="date">
				<span class="update">更新:` + project.PushedAt.Format("2006年1月2日") + `</span>
				<span class="create">创建:` + project.CreatedAt.Format("2006年1月2日") + `</span>
			</div>
			<figcaption>
				<h2>` + project.Name + `</h2>
				<p>` + project.Description + `</p>
				<a href="` + project.URL + `" target="_blank" rel="noreferrer nofollow">GitHub</a>
				<a href="` + project.Homepage + `" target="_blank">Read More</a>
			</figcaption>
		</figure>`
	}

	baseTemplate = bytes.Replace(baseTemplate, []byte("<!-- project list here -->"), []byte(template), 1)
	baseTemplate = bytes.ReplaceAll(baseTemplate, []byte(`<a href="" target="_blank">Read More</a>`), []byte(""))

	os.MkdirAll("public", os.ModePerm)
	os.WriteFile("public/index.html", baseTemplate, os.ModePerm)
}

func mergeProjectData() (allProjects []Project) {
	files, err := os.ReadDir(CACHE_DIR)
	if err != nil {
		panic(err)
	}

	ignoreList := getIgnoreList()
	forks := getForks()

	for _, file := range files {
		buf, err := os.ReadFile(fmt.Sprintf("cache/%s", file.Name()))
		if err != nil {
			panic(err)
		}
		var projects []Project
		json.Unmarshal(buf, &projects)

		for pId := range projects {
			if !projects[pId].Private {
				if slices.Contains(forks, projects[pId].Name) || !projects[pId].Fork {
					if !slices.Contains(ignoreList, projects[pId].Name) {
						allProjects = append(allProjects, projects[pId])
					}
				}
			}
		}
	}
	return allProjects
}

func getForks() (forks []string) {
	buf, err := os.ReadFile("forks.json")
	if err != nil {
		panic(err)
	}
	json.Unmarshal(buf, &forks)
	return forks
}

func getIgnoreList() (ignoreList []string) {
	buf, err := os.ReadFile("ignore.json")
	if err != nil {
		panic(err)
	}
	json.Unmarshal(buf, &ignoreList)
	return ignoreList
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
