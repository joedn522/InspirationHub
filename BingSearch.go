package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const keywordDir = "KeyWordExtract"           
const searchResultDir = "ArticleFetching" 


// Function to read the API key from a file
func readAPIKeyFromFile(filename string) (string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return "", err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    if scanner.Scan() {
        return scanner.Text(), nil
    }

    if err := scanner.Err(); err != nil {
        return "", err
    }

    return "", fmt.Errorf("file is empty")
}

// Function to perform Bing search and return top 3 URLs
func searchBing(query string, bingAPIKey string) ([]string, error) {
	searchURL := "https://api.bing.microsoft.com/v7.0/search"
	client := &http.Client{}

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("q", query)
	q.Add("count", "3") // 获取前3个结果
	q.Add("setLang", "zh-tw") // 设置语言为繁体中文
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Ocp-Apim-Subscription-Key", bingAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	var urls []string
	if webPages, ok := result["webPages"].(map[string]interface{}); ok {
		if value, ok := webPages["value"].([]interface{}); ok {
			for _, v := range value {
				if item, ok := v.(map[string]interface{}); ok {
					if url, ok := item["url"].(string); ok {
						if len(urls) < 3 {
							urls = append(urls, url)
						} else {
							break
						}
					}
				}
			}
		}
	}

	return urls, nil
}

// Function to process each metadata file and perform Bing search
func processMetadataFiles(bingAPIKey string) error {
	fmt.Println("Starting to process metadata files...")
	err := os.MkdirAll(searchResultDir, os.ModePerm)
	if err != nil {
		return err
	}

	err = filepath.Walk(keywordDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".metadata") {
			fmt.Printf("Processing metadata file: %s\n", info.Name())

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			lines := strings.Split(string(content), "\n")
			if len(lines) > 1 {
				var keywords []string
				for _, line := range lines[1:] {
					cleaned := strings.TrimSpace(strings.TrimPrefix(line, "-"))
					if cleaned != "" {
						keywords = append(keywords, cleaned)
					}
				}
				query := strings.Join(keywords, " ")
				query = strings.TrimSpace(query)
				fmt.Printf("Searching Bing for: %s\n", query)

				if query != "" {
					results, err := searchBing(query, bingAPIKey)
					if err != nil {
						return err
					}

					outputFileName := filepath.Join(searchResultDir, strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))+".txt")
					err = ioutil.WriteFile(outputFileName, []byte(strings.Join(results, "\n")), 0644)
					if err != nil {
						return err
					}

					fmt.Printf("Saved search results to: %s\n", outputFileName)
				}
			}
		}

		return nil
	})

	return err
}

func main() {
	bingAPIKey, err := readAPIKeyFromFile("bingkey")
	if err != nil {
		fmt.Printf("Error reading Bing API key: %v\n", err)
		return
	}

	// Use the openaiAPIKey variable as needed
	fmt.Println("Bing API Key:", bingAPIKey)

	err = processMetadataFiles(bingAPIKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
