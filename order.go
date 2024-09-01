package tax_receipt

import (
	"fmt"
	"strconv"
	"time"

	"strings"
)

type Order struct {
	Tipe          string
	OrderDate     time.Time
	OrderNo       string
	ServiceType   string
	Lines         map[string]float64
	TotalCustomer float64
	TotalDriver   float64
}

var (
	DATETIME_FORMAT = Getenv("DATETIME_FORMAT", "2006-01-02")
	HEADER          = strings.Split(Getenv("CSV_FILE_HEADER", ""), ",")
	CUSTOMER_FIELD  = strings.Split(Getenv("CUSTOMER_FIELD", "platform_fee,surcharge,convinience_fee,insurance_fee,admin_fee"), ",")
	PARTNER_FIELD   = strings.Split(Getenv("PARTNER_FIELD", "app_fee,driver_fare,convinience_fee,insurance_fee,admin_fee"), ",")
)

func newOrder(line string) Order {

	lines := strings.Split(line, FIELD_DEMILITER)

	// log.Printf("[%s]\n%d:%v \n%d: %v \n=========", FIELD_DEMILITER, len(HEADER), HEADER, len(lines), lines)

	var order = Order{}
	order.Tipe = lines[0]
	order.OrderDate, err = time.Parse(DATETIME_FORMAT, lines[1])
	order.OrderNo = lines[2]
	order.ServiceType = lines[3]

	// order.PaymentMethod = lines[4]
	// order.Status = lines[5]

	// var total float64 = 0.0
	order.Lines = make(map[string]float64)
	for i := 4; i < len(lines)-2; i++ {
		val := 0.0
		item := HEADER[i-1]
		val, _ = strconv.ParseFloat(strings.Trim(lines[i], " "), 64)
		// if err != nil {
		// 	fmt.Printf("Error Lines: %v : %s :%s\n", err.Error(), item, lines[i])
		// }
		order.Lines[item] = val

	}
	tc, err := strconv.ParseFloat(strings.Trim(lines[len(lines)-2], " "), 64)
	if err != nil {
		// fmt.Printf("Error TC: %v \n %s", err.Error(), lines[len(lines)-2])
		tc = 0.0
	}

	order.TotalCustomer = tc

	td, err := strconv.ParseFloat(strings.Trim(lines[len(lines)-1], " "), 64)
	if err != nil {
		// fmt.Printf("Error TD: %v \n %s", err.Error(), lines[len(lines)-1])
		td = 0.0
	}

	order.TotalDriver = td

	return order
}

func (order Order) IsValid() bool {
	if order.Tipe == "driver" && order.TotalDriver == 0 {
		return false
	}
	if order.Tipe == "customer" && order.TotalCustomer == 0 {
		return false
	}
	return true
}

func (order Order) getReceiptPath(companyId string) string {
	OrderNo := order.OrderNo
	dashed := strings.Split(OrderNo, "-")
	if len(dashed) > 2 {
		OrderNo = fmt.Sprintf("%s/%s", dashed[1], OrderNo)
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s.pdf",
		RECEIPT_PREFIX,
		companyId,
		order.ServiceType,
		order.Tipe,
		order.OrderDate.Format("200601"),
		order.OrderDate.Format("20060102"),
		OrderNo)
}
