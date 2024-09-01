package tax_receipt

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	_ "net/http/pprof"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/googleapis/gax-go/v2"
	"github.com/googleapis/google-cloudevents-go/cloud/storagedata"
	"github.com/zenthangplus/goccm"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	RESULT_BUCKET   = Getenv("RESULT_BUCKET", "gojek_tax_receipts")
	ASSET_FOLDER    = Getenv("ASSET_FOLDER", ".")
	FIELD_DEMILITER = Getenv("FIELD_DEMILITER", ",")
	RETRY_TIMEOUT   = 60
	QUEUE_PATH      string

	MAX_CONCURRENT int
	err            error
	client         *storage.Client
	bucket         *storage.BucketHandle
)

func init() {
	MAX_CONCURRENT, err = strconv.Atoi(Getenv("MAX_CONCURRENT", "100"))
	RETRY_TIMEOUT, _ = strconv.Atoi(Getenv("RETRY_TIMEOUT", "60"))

	if client, err = storage.NewClient(context.Background()); err != nil {
		log.Printf("GCS CONNECTION FAILED storage.Client: %v\n", err)
	}
	bucket = client.Bucket(RESULT_BUCKET)
	// log.Printf("GCS CONNECTED:%+v", bucket)

	QUEUE_PATH = Getenv("QUEUE_PATH", "projects/gjk-fat-int-3r/locations/asia-southeast2/queues/failed-receipt-retrigger")
	functions.HTTP("ReceiptRequest", ReceiptRequest)
	// functions.CloudEvent("ReceiptEvent", ReceiptEvent)
}

func ReceiptRequest(res http.ResponseWriter, req *http.Request) {

	contentType := req.Header.Get("Content-type")
	if contentType == "" {
		contentType = "application/json"
	}

	query := req.URL.Query()
	companyId := query.Get("companyId")
	if companyId == "" {
		companyId = "ID01"
	}
	tipe := query.Get("Tipe")

	// log.Printf("receiving %s request for %s:%s from %s", contentType, companyId, tipe, req.RemoteAddr)
	var results []string

	if strings.Contains(contentType, "application/json") {

		results = processJSONRequest(&req.Body, companyId, tipe)

	} else {

		if strings.Contains(contentType, "text/tsv") {
			FIELD_DEMILITER = string('\t')
		}

		results = processCSVRequest(&req.Body, companyId, tipe)
	}

	log.Printf("Total %d files ; \n", len(results))
	res.WriteHeader(http.StatusOK)
	res.Header().Set("Content-Type", "text/plain") // this
	res.Write([]byte(strings.Join(results, "\n")))

	runtime.GC()
}

func ReceiptEvent(ctx context.Context, event event.Event) error {
	// meta, _ := metadata.FromContext(ctx)
	// log.Printf("ctx meta:%v", meta)
	// expiration := meta.Timestamp.Add(60 * time.Second)
	// if time.Now().After(expiration) {
	// 	log.Printf("event timeout: halting retries for expired event '%v'", meta.EventID)
	// 	return nil
	// }

	sourceClient, _ := storage.NewClient(ctx)
	var data storagedata.StorageObjectData

	if err := protojson.Unmarshal(event.Data(), &data); err != nil {
		log.Printf("protojson.Unmarshal: %v", err)
	}

	paths := strings.Split(data.GetName(), "/")
	companyId := paths[1]
	contentType := data.GetContentType()
	tipe := ""

	srcObj := sourceClient.Bucket(data.GetBucket()).Object(data.GetName())
	objReader, _ := srcObj.NewReader(ctx)
	srcReader := io.NopCloser(objReader)

	log.Printf("receiving %s event for %s:%s from %s", contentType, companyId, tipe, data.GetName())
	var results []string

	if strings.Contains(contentType, "json") {

		results = processJSONRequest(&srcReader, companyId, tipe)

	} else {

		if strings.Contains(contentType, "tsv") {
			FIELD_DEMILITER = string('\t')
		}

		results = processCSVRequest(&srcReader, companyId, tipe)
	}

	log.Printf("Total %d files ; \n", len(results))

	runtime.GC()
	return nil
}

func processCSVRequest(source *io.ReadCloser, companyId string, tipe string) []string {

	c := goccm.New(MAX_CONCURRENT)

	reader := bufio.NewReader(*source)

	var results []string
	firstLine, _, _ := reader.ReadLine()
	HEADER = strings.Split(string(firstLine), FIELD_DEMILITER)

	tipes := []string{"customer", "driver"}
	if tipe != "" {
		tipes = []string{tipe}
	}

	company := getCompany(companyId)

	for {
		byteline, _, err := reader.ReadLine()

		if err == io.EOF {
			break
		}
		c.Wait()
		go func(line string) {

			defer func() {
				if panicInfo := recover(); panicInfo != nil {
					log.Printf("in loop csv:%v:%s\n", panicInfo, line)
				}
			}()

			for _, tpe := range tipes {
				order := newOrder(tpe + FIELD_DEMILITER + line)

				if order.IsValid() {

					receipt := newReceipt(company, order)

					res := ReceiptRender(receipt)
					if res != "" {
						results = append(results, res)
					}
				}
			}
			c.Done()
		}(string(byteline))
	}

	c.WaitAllDone()
	log.Printf("Total %d files ; \n", len(results))
	return results
}

func processJSONRequest(source *io.ReadCloser, companyId string, tipe string) []string {
	c := goccm.New(MAX_CONCURRENT)

	company := getCompany(companyId)

	orders := []Order{}
	json.NewDecoder(*source).Decode(&orders)
	// log.Printf("Orders:%+v\n", orders)

	var results []string
	tipes := []string{"customer", "driver"}
	if tipe != "" {
		tipes = []string{tipe}
	}

	for _, order := range orders {

		for _, tipe := range tipes {
			order.Tipe = tipe
			if !order.IsValid() {
				continue
			}
			c.Wait()
			go func(order Order) {
				// log.Printf("Order:%+v\n", order)

				defer func() {
					if panicInfo := recover(); panicInfo != nil {
						log.Printf("in loop json :%v:%+v\n", panicInfo, order)
					}
				}()

				receipt := newReceipt(company, order)

				res := ReceiptRender(receipt)
				if res != "" {
					results = append(results, res)
				}
				c.Done()
			}(order)
		}
	}

	c.WaitAllDone()
	// source.Close()
	return results
}

func ReceiptRender(receipt *Receipt) string {

	receiptPath := receipt.order.getReceiptPath(receipt.company.Id)

	lineCtx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(RETRY_TIMEOUT*3))

	defer cancel()

	receiptFile := bucket.Object(receiptPath).Retryer(
		// Use WithBackoff to control the timing of the exponential backoff.
		storage.WithBackoff(gax.Backoff{
			// Set the initial retry delay to a maximum of 2 seconds. The length of
			// pauses between retries is subject to random jitter.
			Initial: 5 * time.Second,
			// Set the maximum retry delay to 60 seconds.
			Max: time.Duration(RETRY_TIMEOUT) * time.Second,
			// Set the backoff multiplier to 3.0.
			Multiplier: 3,
		}),
		// Use WithPolicy to customize retry so that all requests are retried even
		// if they are non-idempotent.
		storage.WithPolicy(storage.RetryAlways),
	)
	if receiptFile == nil {
		log.Printf("target receipt file initialization fail")
		return ""
	}

	// time consuming, send to GCS in background
	receiptWriter := receiptFile.NewWriter(lineCtx)
	if err != nil {
		log.Printf("NewWriter Error: %s\n", err.Error())
		return ""
	}

	if receiptWriter == nil {
		log.Printf("GCS File writer not connected")
		return ""
	}

	receipt.render(receiptWriter)

	if err := receiptWriter.Close(); err != nil {
		defer cancel()
		log.Printf("Error on writing %s:%s", receiptPath, err)
		QueueFailedUpload(receipt)
		return ""
	}

	return receiptPath
}

func QueueFailedUpload(receipt *Receipt) *taskspb.Task {
	gTaskctx := context.Background()
	client, err := cloudtasks.NewClient(gTaskctx)
	if err != nil {
		log.Printf("Tasks NewClient Failed: %w", err)
		return nil
	}
	defer client.Close()

	receiptPath := receipt.order.getReceiptPath(receipt.company.Id)
	gcs_api_url := fmt.Sprintf("https://storage.googleapis.com/upload/storage/v1/b/%s/o?uploadType=media&name=%s", RESULT_BUCKET, receiptPath)
	pdfByte := receipt.render(nil)
	req := &taskspb.CreateTaskRequest{
		Parent: QUEUE_PATH,
		Task: &taskspb.Task{
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        gcs_api_url,
					AuthorizationHeader: &taskspb.HttpRequest_OauthToken{
						OauthToken: &taskspb.OAuthToken{
							ServiceAccountEmail: Getenv("SERVICE_ACCOUNT", "gjk-fat-int-3r@appspot.gserviceaccount.com"),
						},
					},
					Headers: map[string]string{"Content-Type": "application/pdf", "Content-Size": fmt.Sprint(len(pdfByte))},
					Body:    pdfByte,
				},
			},
		},
	}

	task, err := client.CreateTask(gTaskctx, req)
	if err != nil {
		log.Printf("Failed to requeue GCS Upload cloudtasks.CreateTask: %v ", err)
	}

	return task

}
