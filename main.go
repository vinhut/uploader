package main

import (
	"github.com/gin-gonic/gin"
	opentracing "github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"

	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

var SERVICE_NAME = "uploader-service"
var APIKEY = os.Getenv("APIKEY")

type ImageService struct {
	Data    Data `json:"data"`
	Success bool `json:"success"`
	Status  int  `json:"status"`
}
type Image struct {
	Filename  string `json:"filename"`
	Name      string `json:"name"`
	Mime      string `json:"mime"`
	Extension string `json:"extension"`
	URL       string `json:"url"`
	Size      int    `json:"size"`
}
type Thumb struct {
	Filename  string `json:"filename"`
	Name      string `json:"name"`
	Mime      string `json:"mime"`
	Extension string `json:"extension"`
	URL       string `json:"url"`
	Size      string `json:"size"`
}
type Medium struct {
	Filename  string `json:"filename"`
	Name      string `json:"name"`
	Mime      string `json:"mime"`
	Extension string `json:"extension"`
	URL       string `json:"url"`
	Size      string `json:"size"`
}
type Data struct {
	ID         string `json:"id"`
	URLViewer  string `json:"url_viewer"`
	URL        string `json:"url"`
	DisplayURL string `json:"display_url"`
	Title      string `json:"title"`
	Time       string `json:"time"`
	Image      Image  `json:"image"`
	Thumb      Thumb  `json:"thumb"`
	Medium     Medium `json:"medium"`
	DeleteURL  string `json:"delete_url"`
}

func setupRouter() *gin.Engine {

	var JAEGER_COLLECTOR_ENDPOINT = os.Getenv("JAEGER_COLLECTOR_ENDPOINT")
	cfg := jaegercfg.Configuration{
		ServiceName: "uploader-service",
		Sampler: &jaegercfg.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:          true,
			CollectorEndpoint: JAEGER_COLLECTOR_ENDPOINT,
		},
	}
	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory
	tracer, _, _ := cfg.NewTracer(
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
	)
	opentracing.SetGlobalTracer(tracer)

	router := gin.Default()

	router.GET(SERVICE_NAME+"/ping", func(c *gin.Context) {
		c.String(200, "OK")
	})

	router.POST(SERVICE_NAME+"/upload", func(c *gin.Context) {

		span := tracer.StartSpan("upload")

		result, upload_err := c.FormFile("uploadfile")
		if upload_err != nil {
			panic(upload_err)
		}
		file, open_error := result.Open()
		if open_error != nil {
			panic(open_error)
		}
		buff := make([]byte, result.Size)
		_, read_err := file.Read(buff)
		if read_err != nil {
			panic(read_err)
		}
		encoded := base64.StdEncoding.EncodeToString(buff)

		link := "https://api.imgbb.com/1/upload?key=" + APIKEY
		v := url.Values{}
		v.Add("image", encoded)
		resp, post_err := http.PostForm(link, v)

		defer resp.Body.Close()
		if post_err != nil {
			panic(post_err)
		}
		imageResponse := ImageService{}
		if resp.StatusCode == 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			json.Unmarshal([]byte(body), &imageResponse)
			image_url := imageResponse.Data.URL
			c.String(200, image_url)
			span.Finish()
		} else {
			c.String(403, "not uploaded")
			span.Finish()
		}

	})

	return router

}

func main() {

	router := setupRouter()
	router.Run(":8080")

}
