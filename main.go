// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"signInWithApple/actions"
)

const AK = "AKIAUW65UPQ75PP2QUEF"                // AWS AK
const SK = ""                                    // AWS SK
const BUCKET = "creativi-kids-server-testing-01" // AWS BUCKET Name
const REGION = "ap-southeast-2"
const uploadFilename = "/home/docker/uwsgi-2.0.23.tar.gz"
const uploadKey = "books/203b53f7-bd06-4a17-929e-c157a822b52a/pictures/uwsgi-2.0.23.tar.gz"

var client *s3.Client

type Credential struct {
	Ak string
	Sk string
}

func (c *Credential) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID: c.Ak, SecretAccessKey: c.Sk,
	}, nil
}

// snippet-start:[gov2.s3.IHttpRequester.helper]

// IHttpRequester abstracts HTTP requests into an interface so it can be mocked during
// unit testing.
type IHttpRequester interface {
	Get(url string) (resp *http.Response, err error)
	Put(url string, contentLength int64, body io.Reader) (resp *http.Response, err error)
	Delete(url string) (resp *http.Response, err error)
}

// HttpRequester uses the net/http package to make HTTP requests during the scenario.
type HttpRequester struct{}

func (httpReq HttpRequester) Get(url string) (resp *http.Response, err error) {
	return http.Get(url)
}
func (httpReq HttpRequester) Put(url string, contentLength int64, body io.Reader) (resp *http.Response, err error) {
	putRequest, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}
	putRequest.ContentLength = contentLength
	return http.DefaultClient.Do(putRequest)
}
func (httpReq HttpRequester) Delete(url string) (resp *http.Response, err error) {
	delRequest, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(delRequest)
}

type PictureTagging struct {
	PageNo int    `json:"pageNo"`
	Desc   string `json:"desc"`
	Prompt string `json:"prompt"`
}

// snippet-end:[gov2.s3.IHttpRequester.helper]

// snippet-start:[gov2.s3.Scenario_Presigning]

// RunPresigningScenario is an interactive example that shows you how to get presigned
// HTTP requests that you can use to move data into and out of Amazon Simple Storage
// Service (Amazon S3). The presigned requests contain temporary credentials and can
// be used by an HTTP client.
//
// 1. Get a presigned request to put an object in a bucket.
// 2. Use the net/http package to use the presigned request to upload a local file to the bucket.
// 3. Get a presigned request to get an object from a bucket.
// 4. Use the net/http package to use the presigned request to download the object to a local file.
// 5. Get a presigned request to delete an object from a bucket.
// 6. Use the net/http package to use the presigned request to delete the object.
//
// This example creates an Amazon S3 presign client from the specified sdkConfig so that
// you can replace it with a mocked or stubbed config for unit testing.
//
// It uses a questioner from the `demotools` package to get input during the example.
// This package can be found in the ..\..\demotools folder of this repo.
//
// It uses an IHttpRequester interface to abstract HTTP requests so they can be mocked
// during testing.
func RunPresigningScenario(sdkConfig aws.Config, httpRequester IHttpRequester) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Something went wrong with the demo.")
		}
	}()

	log.Println(strings.Repeat("-", 88))
	log.Println("Welcome to the Amazon S3 presigning demo.")
	log.Println(strings.Repeat("-", 88))

	s3Client := s3.NewFromConfig(sdkConfig)
	bucketBasics := actions.BucketBasics{S3Client: s3Client}
	presignClient := s3.NewPresignClient(s3Client)
	presigner := actions.Presigner{PresignClient: presignClient}

	bucketName := BUCKET
	bucketExists, err := bucketBasics.BucketExists(bucketName)
	if err != nil {
		panic(err)
	}
	if !bucketExists {
		err = bucketBasics.CreateBucket(bucketName, sdkConfig.Region)
		if err != nil {
			panic(err)
		} else {
			log.Println("Bucket created.")
		}
	}
	log.Println(strings.Repeat("-", 88))

	log.Printf("Let's presign a request to upload a file to your bucket.")
	uploadFile, err := os.Open(uploadFilename)
	if err != nil {
		panic(err)
	}
	defer uploadFile.Close()
	presignedPutRequest, err := presigner.PutObject(bucketName, uploadKey, 60)
	if err != nil {
		panic(err)
	}
	log.Printf("Got a presigned %v request to URL:\n\t%v\n", presignedPutRequest.Method,
		presignedPutRequest.URL)

	log.Println("Using net/http to send the request...")
	info, err := uploadFile.Stat()
	if err != nil {
		panic(err)
	}
	putResponse, err := httpRequester.Put(presignedPutRequest.URL, info.Size(), uploadFile)
	if err != nil {
		panic(err)
	}
	log.Printf("%v object %v with presigned URL returned %v.", presignedPutRequest.Method,
		uploadKey, putResponse.StatusCode)
	log.Println(strings.Repeat("-", 88))

	log.Println("Let's tag the object we just uploaded.")
	taggings := make(map[string]string)
	taggings["pageNo"] = "12345"
	taggings["prompt"] = "test with updated round2"
	taggings["desc"] = "test with updated round2"
	err = bucketBasics.PutObjectTaggings(bucketName, uploadKey, taggings)

	log.Printf("Let's presign a request to download the object.")
	presignedGetRequest, err := presigner.GetObject(bucketName, uploadKey, 60)
	if err != nil {
		panic(err)
	}
	log.Printf("Got a presigned %v request to URL:\n\t%v\n", presignedGetRequest.Method,
		presignedGetRequest.URL)
	log.Println("Using net/http to send the request...")
	getResponse, err := httpRequester.Get(presignedGetRequest.URL)
	if err != nil {
		panic(err)
	}
	log.Printf("%v object %v with presigned URL returned %v.", presignedGetRequest.Method,
		uploadKey, getResponse.StatusCode)
	defer getResponse.Body.Close()
	downloadBody, err := io.ReadAll(getResponse.Body)
	if err != nil {
		panic(err)
	}
	log.Printf("Downloaded %v bytes. Here are the first 100 of them:\n", len(downloadBody))
	log.Println(strings.Repeat("-", 88))
	log.Println(string(downloadBody[:100]))
	log.Println(strings.Repeat("-", 88))

	//log.Println("Let's presign a request to delete the object.")
	//presignedDelRequest, err := presigner.DeleteObject(bucketName, uploadKey)
	//if err != nil {
	//	panic(err)
	//}
	//log.Printf("Got a presigned %v request to URL:\n\t%v\n", presignedDelRequest.Method,
	//	presignedDelRequest.URL)
	//log.Println("Using net/http to send the request...")
	//delResponse, err := httpRequester.Delete(presignedDelRequest.URL)
	//if err != nil {
	//	panic(err)
	//}
	//log.Printf("%v object %v with presigned URL returned %v.\n", presignedDelRequest.Method,
	//	uploadKey, delResponse.StatusCode)
	log.Println(strings.Repeat("-", 88))

	log.Println("Thanks for watching!")
	log.Println(strings.Repeat("-", 88))
}

func main() {
	cred := &Credential{
		Ak: AK,
		Sk: SK,
	}
	// creds , _ := cred.Retrieve(context.Background())

	cfg := aws.Config{
		Region:                      REGION,
		Credentials:                 cred,
		BearerAuthTokenProvider:     nil,
		HTTPClient:                  nil,
		EndpointResolverWithOptions: nil,
		RetryMaxAttempts:            0,
		RetryMode:                   "",
		Retryer:                     nil,
		ConfigSources:               nil,
		APIOptions:                  nil,
		Logger:                      nil,
		ClientLogMode:               0,
		DefaultsMode:                "",
		RuntimeEnvironment:          aws.RuntimeEnvironment{},
	}

	client = s3.NewFromConfig(cfg)
	RunPresigningScenario(cfg, HttpRequester{})
}
