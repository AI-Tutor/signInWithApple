package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type ImageData struct {
	ImageName string `form:"imageName"`
	ImageMD5  string `form:"imageMD5"`
}

type PreSignURLResp struct {
	ImageName     string `form:"imageName"`
	ImageMD5      string `form:"imageMD5"`
	PutPreSignURL string `form:""`
}

func init() {
	cred := &Credential{
		Ak: AK,
		Sk: SK,
	}

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
}

func uploadIMageFile(filePath string, preSignURL string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Failed to open file", err)
		return
	}
	defer file.Close()

	httpClient := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, preSignURL, file)
	if err != nil {
		fmt.Println("Failed to create HTTP request", err)
		return
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Println("Failed to send HTTP request", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response status code:", resp)
	if resp.StatusCode == http.StatusOK {
		fmt.Println("File uploaded successfully")
	} else {
		fmt.Println("Failed to upload file")
	}
}

func AWSV2PresignPutObject(file ImageData) PreSignURLResp {

	fmt.Println("Create Presign client")
	presignClient := s3.NewPresignClient(client)

	// filemd5 := "/SX1AAfPuVitH7ZK9bNg6Q==" // 前端上传文件给到 AWS-S3 时文件MD5校验（可选）
	// filename := "/demo/test.jpg"          // 文件存储bucket的路径和名称

	presignResult, err := presignClient.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(BUCKET),
		Key:    aws.String(file.ImageName),
		ACL:    types.ObjectCannedACLPrivate,
		//ContentMD5: &file.ImageMD5,
	}, func(po *s3.PresignOptions) {
		// 授权时效
		po.Expires = 10 * time.Minute
	})

	if err != nil {
		panic("Couldn't get presigned URL for GetObject")
	}
	fmt.Printf("上传URL: %s\n", presignResult.URL)

	uploadIMageFile("/home/ubuntu/WechatIMG902.jpg", presignResult.URL)
	//presignResult, err = presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
	//	Bucket: aws.String(BUCKET),
	//	Key:    aws.String(file.ImageName),
	//}, func(po *s3.PresignOptions) {
	//	// 授权时效
	//	po.Expires = 48 * time.Hour
	//})
	//
	//if err != nil {
	//	panic("Couldn't get presigned URL for GetObject")
	//}
	//fmt.Printf("访问URL: %s\n", presignResult.URL)
	return PreSignURLResp{
		ImageName:     file.ImageName,
		ImageMD5:      file.ImageMD5,
		PutPreSignURL: presignResult.URL,
	}
}

func fetchPreSignResult(c *gin.Context) {
	var imageData ImageData
	var resp PreSignURLResp
	if c.ShouldBind(&imageData) == nil {
		resp = AWSV2PresignPutObject(imageData)
	}
	c.IndentedJSON(http.StatusOK, resp)
}

//
//func main() {
//	router := gin.Default()
//	router.GET("/api/v1/preSignURL", fetchPreSignResult)
//	router.Run(":8085")
//}

// CGO_ENABLED=1 GOOS=linux  GOARCH=amd64  CC=x86_64-linux-musl-gcc  CXX=x86_64-linux-musl-g++ go build main.go
