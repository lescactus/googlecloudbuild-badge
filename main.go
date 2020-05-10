// Package p contains a Pub/Sub Cloud Function.
package p

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"cloud.google.com/go/storage"
)

const (
	bucket    = "gcp-build-badge"
	gitrepo   = "http-gallery-beego"
	svgPath   = gitrepo + "/"
	statusSVG = svgPath + "statusbadge.svg"
)

// PubSubMessage is the payload of a Pub/Sub event. Please refer to the docs for
// additional information regarding Pub/Sub events.
type PubSubMessage struct {
	Data []byte `json:"data"`
}

// StatusCloudBuild consumes a Pub/Sub message.
func StatusCloudBuild(ctx context.Context, m PubSubMessage) error {
	// Unmarshal the PubSub json data message
	var message map[string]interface{}
	json.Unmarshal([]byte(m.Data), &message)

	// We just need the build status and git repo & branch
	status := message["status"]
	repo := message["substitutions"].(map[string]interface{})["REPO_NAME"]
	branch := message["substitutions"].(map[string]interface{})["BRANCH_NAME"]

	log.Printf("status: %v\n", status)
	log.Printf("repo: %v\n", repo)
	log.Printf("branch: %v\n", branch)

	// Modify the status badge on SUCCESS or FAILURE builds only
	if repo == gitrepo && (status == "SUCCESS" || status == "FAILURE") {
		client, err := storage.NewClient(ctx)
		if err != nil {
			log.Fatalln(err)
			return err
		}
		log.Printf("Going to copy %s into %s...\n", string(svgPath+strings.ToLower(status.(string))+".svg"), statusSVG)

		src := client.Bucket(bucket).Object(string(svgPath + strings.ToLower(status.(string)) + ".svg"))
		dst := client.Bucket(bucket).Object(statusSVG)

		copier := dst.CopierFrom(src)
		copier.ContentType = "image/svg+xml"
		copier.CacheControl = "no-cache, max-age=0" // Github has a cache macanism. Adding the Cache-Control: "no-cache, max-ago=0" HTTP header
		_, err = copier.Run(ctx)                    // prevent the status badge from being cached.
		if err != nil {                             // See: https://github.com/github/markup/issues/224#issuecomment-48532178
			log.Fatalln(err)
			return err
		}
		log.Println("Done")
		return nil
	}
	return nil
}
