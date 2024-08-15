package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const bingAPIKey = "" // 替换为你的 Bing API Key
const metaDir = "metaDir"            // 替换为你生成的 metadata 目录
const searchResultDir = "ArticleFetching" // 保存搜索结果的目录

// Function to perform Bing search and return top 3 URLs
func searchBing(query string) ([]string, error) {
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
func processMetadataFiles() error {
	fmt.Println("Starting to process metadata files...")
	err := os.MkdirAll(searchResultDir, os.ModePerm)
	if err != nil {
		return err
	}

	err = filepath.Walk(metaDir, func(path string, info os.FileInfo, err error) error {
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
					results, err := searchBing(query)
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
	err := processMetadataFiles()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
