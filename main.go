///bin/sh -c true; exec /usr/bin/env go run "$0" "$@"
//  vim:ts=4:sts=4:sw=4:noet
//
//  Author: Hari Sekhon
//  Date: 2024-10-02 05:17:53 +0300 (Wed, 02 Oct 2024)
//
//  https///github.com/HariSekhon/GitHub-Repos-MermaidJS-Gantt-Chart
//
//  License: see accompanying Hari Sekhon LICENSE file
//
//  If you're using my code you're welcome to connect with me on LinkedIn and optionally send me feedback to help steer this or other code I publish
//
//  https://www.linkedin.com/in/HariSekhon
//

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type GitHubRepo struct {
    Name      string    `json:"name"`
    Fork      bool      `json:"fork"`
    CreatedAt time.Time `json:"created_at"`
    PushedAt  time.Time `json:"pushed_at"`
}

type GitHubCommit struct {
	Commit struct {
		Author struct {
			Date time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

func main() {
	helpFlag := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *helpFlag {
		fmt.Println(`
Generates a Mermaid.js Gantt chart of a GitHub user's public repos active dates
using each created and pushed date

Arguments:
  <github_username>      GitHub username for which to fetch the repositories

Environment Variables:
  GH_TOKEN               GitHub token (preferred as it matches GitHub CLI environment variable)
  GITHUB_TOKEN           Fallback GitHub token (if GH_TOKEN is not set)

Usage: go run main.go <github_username>
`)
		os.Exit(3)
	}

	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <github_username>")
	}
	username := os.Args[1]

	githubToken := os.Getenv("GH_TOKEN")
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}
	if githubToken == "" {
		log.Fatal("GitHub token not found in environment variables (GH_TOKEN or GITHUB_TOKEN)")
	}

	repos, err := fetchRepos(username, githubToken)
	if err != nil {
		log.Fatalf("Error fetching repos: %v\n", err)
	}

    log.Info("Generating Gantt Chart")
	ganttChart := generateGanttChart(repos)

	initFile := "init.mmd"
	log.Info("Reading Gantt Chart Config from ", initFile)
	ganttConfigBytes, err := ioutil.ReadFile(initFile)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	ganttConfig := string(ganttConfigBytes)

	filename := "gantt_chart.mmd"
	log.Info("Writing to ", filename)
	err = writeGanttChartToFile(ganttConfig + ganttChart, filename)
	if err != nil {
		log.Fatalf("Error writing Gantt chart to file: %v", err)
	}

	log.Info("Markdown file with Mermaid.js Gantt chart generated successfully")
}

func fetchRepos(username, token string) ([]GitHubRepo, error) {
	var allRepos []GitHubRepo
	page := 1

	log.Info("Fetching public GitHub repos for user: ", username)

	for {
		url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100&page=%d", username, page)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "token " + token)

		client := &http.Client{}
		log.Info("Fetching GitHub repos page: ", page)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var repos []GitHubRepo
		err = json.Unmarshal(body, &repos)
		if err != nil {
			return nil, err
		}

		if len(repos) == 0 {
			break
		}

        // filter out forked repos
        for _, repo := range repos {
            if !repo.Fork {
                allRepos = append(allRepos, repo)
            }
        }

		if resp.Header.Get("Link") == "" || ! hasNextPage(resp.Header.Get("Link")) {
			break
		}

		page++
	}

	return allRepos, nil
}

func hasNextPage(linkHeader string) bool {
	// The Link header contains links to the next, previous, first, and last pages of results
	// Example: <https://api.github.com/user/repos?page=2>; rel="next",
	//          <https://api.github.com/user/repos?page=34>; rel="last"
	return strings.Contains(linkHeader, `rel="next"`)
}

func fetchFirstAndLastCommit(owner, repo, token string) (time.Time, time.Time, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits", owner, repo)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token " + token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	var commits []GitHubCommit
	err = json.Unmarshal(body, &commits)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	// first commit is the last in the array (reverse chronological)
	firstCommit := commits[len(commits)-1].Commit.Author.Date

	// last commit is the first one in the array
	lastCommit := commits[0].Commit.Author.Date

	return firstCommit, lastCommit, nil
}

func generateGanttChart(repos []GitHubRepo) string {
    // sort the repos by start date (CreatedAt)
    sort.Slice(repos, func(i, j int) bool {
        return repos[i].CreatedAt.Before(repos[j].CreatedAt)
    })

    ganttChart := "gantt\n    dateFormat  YYYY-MM-DD\n    title Repositories Gantt Chart\n"
    for _, repo := range repos {
		taskType := "active"
		if ! isWithinLastSixMonths(repo.PushedAt) {
			taskType = "done"
		}
        ganttChart += fmt.Sprintf("    %s : %s, %s, %s\n",
									repo.Name,
									taskType,
									repo.CreatedAt.Format("2006-01-02"),
									repo.PushedAt.Format("2006-01-02"))
    }

    return ganttChart
}

func writeGanttChartToFile(ganttChart string, fileName string) error {
    // create or overwrite the specified file
    file, err := os.Create(fileName)
    if err != nil {
        return fmt.Errorf("could not create file %s: %v", fileName, err)
    }
    defer file.Close()

    _, err = file.WriteString(ganttChart)
    if err != nil {
        return fmt.Errorf("could not write to file %s: %v", fileName, err)
    }

    return nil
}

func isWithinLastSixMonths(date time.Time) bool {
    now := time.Now()

    // calculate the date that is 6 months ago from now
    sixMonthsAgo := now.AddDate(0, -6, 0)

    // check if the given date is after or equal to the date 6 months ago
    return date.After(sixMonthsAgo)
}
