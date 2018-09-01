package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"darlinggo.co/version"
	"github.com/mitchellh/cli"
)

type impracticalUpdates struct {
	Version    string `json:"version,omitempty"`
	Deprecated bool   `json:"deprecated,omitempty"`
}

type author struct {
	Name   string `json:"name,omitempty"`
	URL    string `json:"url,omitempty"`
	Avatar string `json:"avatar,omitempty"`
}

type attachment struct {
	URL               string `json:"url"`
	MIMEType          string `json:"mime_type"`
	Title             string `json:"title,omitempty"`
	SizeInBytes       int64  `json:"size_in_bytes,omitempty"`
	DurationInSeconds int64  `json:"duration_in_seconds,omitempty"`
}

type item struct {
	ID                 string             `json:"id"`
	URL                string             `json:"url,omitempty"`
	ExternalURL        string             `json:"external_url,omitempty"`
	Title              string             `json:"title,omitempty"`
	ContentHTML        string             `json:"content_html,omitempty"`
	ContentText        string             `json:"content_text,omitempty"`
	Summary            string             `json:"summary,omitempty"`
	Image              string             `json:"image,omitempty"`
	BannerImage        string             `json:"banner_image,omitempty"`
	DatePublished      time.Time          `json:"date_published,omitempty"`
	DateModified       time.Time          `json:"date_modified,omitempty"`
	Author             author             `json:"author,omitempty"`
	Tags               []string           `json:"tags,omitempty"`
	Attachments        []attachment       `json:"attachments,omitempty"`
	ImpracticalUpdates impracticalUpdates `json:"_impractical_updates,omitempty"`
}

type hub struct {
	Type string `json:"type,omitempty"`
	URL  string `json:"url,omitempty"`
}

type feed struct {
	Version     string `json:"version"`
	Title       string `json:"title"`
	HomePageURL string `json:"home_page_url,omitempty"`
	FeedURL     string `json:"feed_url,omitempty"`
	Description string `json:"description,omitempty"`
	NextURL     string `json:"next_url,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Favicon     string `json:"favicon,omitempty"`
	Author      author `json:"author,omitempty"`
	Expired     bool   `json:"expired,omitempty"`
	Hubs        []hub  `json:"hubs,omitempty"`
	Items       []item `json:"items,omitempty"`
}

func fetchFeed(url string) (feed, error) {
	resp, err := http.Get(url)
	if err != nil {
		return feed{}, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return feed{}, err
	}
	var f feed
	err = json.Unmarshal(b, &f)
	if err != nil {
		return feed{}, err
	}
	return f, nil
}

func getVersionsSince(curVersion, url string) ([]item, error) {
	var items []item
	u := url
	for {
		// grab a page
		f, err := fetchFeed(u)
		if err != nil {
			return nil, err
		}

		// check for our version in the page
		found := -1
		for pos, i := range items {
			if i.ImpracticalUpdates.Version == curVersion {
				found = pos
				break
			}
		}
		if found >= 0 {
			// if our version is in the page, only get it and the items
			// before it, then bail out
			items = append(items, f.Items[:found+1]...)
			break
		}

		// add the items to our list
		items = append(items, f.Items...)

		// update our URL to the next page
		u = f.NextURL
		if u == "" {
			// if there is no next page, we're done here
			break
		}
	}
	return items, nil
}

func versionCommandFactory(ui cli.Ui) func() (cli.Command, error) {
	return func() (cli.Command, error) {
		return versionCommand{
			ui:          ui,
			curVersion:  version.Tag,
			updatesFeed: "https://gitlab.com/snippets/1735555/raw",
		}, nil
	}
}

type versionCommand struct {
	ui          cli.Ui
	curVersion  string
	updatesFeed string
}

func (v versionCommand) Help() string {
	return `Display the current version of tanglesd, and check for updates.`
}

func (v versionCommand) Run(args []string) int {
	fmt.Printf("tanglesd version %s\n", v.curVersion)
	versions, err := getVersionsSince(v.curVersion, v.updatesFeed)
	if err != nil {
		log.Println("Error checking for updates: %s", err.Error())
		return 0
	}
	if len(versions) < 1 {
		return 0
	}

	fmt.Println("")

	newVersion := versions[0]
	curVersion := versions[len(versions)-1]
	urgent := ""
	recommended := ""
	if curVersion.ImpracticalUpdates.Deprecated {
		urgent = fmt.Sprintf("[!] Your current version of tanglesd, %s, is deprecated. ", curVersion.ImpracticalUpdates.Version)
		recommended = " It is *highly* recommended that you update as soon as possible."
	}
	fmt.Printf("%sA newer version of tanglesd, %s, is available for download now.%s To get it, head to %s.\n", urgent, newVersion.ImpracticalUpdates.Version, recommended, newVersion.URL)
	return 0
}

func (v versionCommand) Synopsis() string {
	return "Display the current version of tanglesd and check if a newer version is available."
}
