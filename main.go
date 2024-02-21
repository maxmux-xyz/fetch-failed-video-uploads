package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

type Video struct {
	VideoLibraryId       int      `json:"videoLibraryId"`
	Guid                 string   `json:"guid"`
	Title                string   `json:"title"`
	DateUploaded         string   `json:"dateUploaded"`
	Views                int      `json:"views"`
	IsPublic             bool     `json:"isPublic"`
	Length               int      `json:"length"`
	Status               int      `json:"status"`
	Framerate            float64  `json:"framerate"`
	Rotation             int      `json:"rotation"`
	Width                int      `json:"width"`
	Height               int      `json:"height"`
	AvailableResolutions string   `json:"availableResolutions"`
	ThumbnailCount       int      `json:"thumbnailCount"`
	EncodeProgress       int      `json:"encodeProgress"`
	StorageSize          int64    `json:"storageSize"`
	Captions             []string `json:"captions"`
	HasMP4Fallback       bool     `json:"hasMP4Fallback"`
	CollectionId         string   `json:"collectionId"`
	ThumbnailFileName    string   `json:"thumbnailFileName"`
	AverageWatchTime     int      `json:"averageWatchTime"`
	TotalWatchTime       int      `json:"totalWatchTime"`
	Category             string   `json:"category"`
	Chapters             []string `json:"chapters"` // Assuming chapters are strings; adjust if it's a complex type
	Moments              []string `json:"moments"`  // Same assumption as for chapters
	MetaTags             []string `json:"metaTags"`
	TranscodingMessages  []string `json:"transcodingMessages"`
}

type BunnyResp struct {
	TotalItems   int     `json:"totalItems"`
	CurrentPage  int     `json:"currentPage"`
	ItemsPerPage int     `json:"itemsPerPage"`
	Items        []Video `json:"items"`
}

var (
	globalList []string // To store values extracted from API responses
	// listMutex  sync.Mutex
)

func getVideoList(page string, itemsPerPage string, saveStatus bool) BunnyResp {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	libraryId := os.Getenv("LIBRARYID")
	accessKey := os.Getenv("ACCESSKEY")
	url := "https://video.bunnycdn.com/library/" + libraryId + "/videos?page=" + page + "&itemsPerPage=" + itemsPerPage + "&orderBy=date"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("accept", "application/json")
	req.Header.Add("AccessKey", accessKey)
	res, _ := http.DefaultClient.Do(req)
	if res.StatusCode != 200 {
		log.Fatalf("Error occurred during API call. Status: %d", res.StatusCode)
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var bunnyResp BunnyResp
	err = json.Unmarshal([]byte(body), &bunnyResp)
	if err != nil {
		log.Fatal(err)
		log.Fatalf("Error occurred during unmarshaling. Error: %s", err.Error())
	}

	if saveStatus {
		for _, video := range bunnyResp.Items {
			if video.Status != 4 {
				globalList = append(globalList, video.Guid)
			}
		}
	}
	return bunnyResp
}

func work(itemsPerPage string, saveStatus bool, wg *sync.WaitGroup, ch chan int) {
	for p := range ch {
		fmt.Println("Fetching page: ", p)
		getVideoList(strconv.Itoa(p), itemsPerPage, saveStatus)
		wg.Done()
	}
}

func main() {
	// Get videos concurrently
	videoPage := getVideoList("1", "1", false)

	// Now I want to calculate how many times I want to call the getVideoList function
	targetItemsPerPage := 100
	numberOfPages := int(math.Ceil(float64(videoPage.TotalItems) / float64(targetItemsPerPage)))
	fmt.Printf("%+v\n", numberOfPages)

	dataChannel := make(chan int)

	const maxNumWorkers = 5
	var wg sync.WaitGroup

	// Start workers
	for i := 1; i <= maxNumWorkers; i++ {
		go work(strconv.Itoa(targetItemsPerPage), true, &wg, dataChannel)
	}

	wg.Add(numberOfPages)
	// Send tasks to workers
	for i := 1; i <= numberOfPages; i++ {
		dataChannel <- i
	}

	close(dataChannel)
	// Wait for all tasks to complete
	wg.Wait()

	fmt.Println("Global List:", globalList)
}
