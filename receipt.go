package tax_receipt

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/signintech/gopdf"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	STORAGE_PREFIX_PATH = Getenv("STORAGE_PREFIX_PATH", "/tmp/")

	LOCALE          = Getenv("LOCALE", "id_ID")
	RECEIPT_PREFIX  = Getenv("RECEIPT_PREFIX", "AKAB/receipts")
	DATE_FORMAT     = Getenv("DATE_FORMAT", "Monday, 02 January 2006")
	CUSTOMER_FIELDS = strings.Split(Getenv("CUSTOMER_FIELD", "revenue_from_platform_fee,surplus_amount,booking_fee_amount,revenue_from_delivery_fare,discount_customer"), ",")
	DRIVER_FIELDS   = strings.Split(Getenv("DRIVER_FIELDS", "revenue_from_driver_fare,discount_customer"), ",")
	monthReplacer   = strings.NewReplacer(
		"January", "Januari",
		"February", "Februari",
		"March", "Maret",
		"April", "April",
		"May", "Mei",
		"June", "Juni",
		"July", "Juli",
		"August", "Agustus",
		"September", "September",
		"October", "Oktober",
		"November", "November",
		"December", "Desember")

	dayReplacer = strings.NewReplacer(
		"Monday", "Senin",
		"Tuesday", "Selasa",
		"Wednesday", "Rabu",
		"Thursday", "Kamis",
		"Friday", "Jumat",
		"Saturday", "Sabtu",
		"Sunday", "Minggu")

	msg *message.Printer

	normalFont []byte
	boldFont   []byte
)

func init() {

	normalFont, err = os.ReadFile(ASSET_FOLDER + "/assets/fonts/cour.ttf")
	boldFont, err = os.ReadFile(ASSET_FOLDER + "/assets/fonts/courbd.ttf")

	langTag := language.Indonesian
	initLanguage(langTag, ASSET_FOLDER+"/assets/id-ID.json")
	msg = message.NewPrinter(langTag)

}

func _t(key string) string {
	return msg.Sprintf(key)
}

func receiptLine(item string, value float64) string {
	return msg.Sprintf("%-30s %18.0f", _t(item), value) //#+ msgPrinter.Sprintf("%20%.0f", value)
}

func center(s string, w int) string {
	s = _t(s)
	return msg.Sprintf("%*s", -w, fmt.Sprintf("%*s", (w+len(s))/2, s))
}

type Receipt struct {
	company Company
	order   Order
	pdf     gopdf.GoPdf
	curLine float64
}

func newReceipt(company Company, order Order) *Receipt {
	r := new(Receipt)
	r.company = company
	r.order = order

	var pageHeight = 160
	// if len(order.lines) > 3 {
	// 	pageHeight = pageHeight + (len(order.lines)-3)*8
	// }

	r.pdf = gopdf.GoPdf{}

	r.pdf.Start(gopdf.Config{PageSize: gopdf.Rect{W: 228.0, H: float64(pageHeight)}, Unit: gopdf.UnitPT})
	r.pdf.AddPage()

	if err := r.pdf.AddTTFFontDataWithOption("Courier", normalFont, gopdf.TtfOption{false, 0, nil, nil}); err != nil {
		fmt.Println(err.Error())
	}
	if err := r.pdf.AddTTFFontDataWithOption("Courier", boldFont, gopdf.TtfOption{false, 2, nil, nil}); err != nil {
		fmt.Println(err.Error())
	}
	r.curLine = 6.0
	return r
}

func (this *Receipt) addLine(text string) {
	lineHeight := 8.0
	this.curLine = this.curLine + lineHeight
	this.pdf.SetX(8.0)
	this.pdf.SetY(this.curLine)
	this.pdf.Text(text)
	// return this
}

func (this *Receipt) printHeader() {
	this.pdf.SetFont("Courier", "B", 7)

	this.addLine(center(this.company.Name, 50))          //, {align:"center"} );
	this.addLine(center("NPWP: "+this.company.NPWP, 50)) //, {align:"center",continued:false});

	this.pdf.SetFont("Courier", "", 7)
	for _, address := range this.company.Address {
		this.addLine(center(address, 50)) //,{align:"center"})
	}

	curY := 50.0
	this.pdf.SetLineWidth(2)
	this.pdf.SetLineType("solid")
	this.pdf.Line(8, curY, 218, curY)
}

func (this *Receipt) printItemLines() {

	total := 0.0
	printed_lines := DRIVER_FIELDS
	if this.order.Tipe == "customer" {
		printed_lines = CUSTOMER_FIELDS
	}
	// log.Printf("lines:%+v", printed_lines)
	for _, item := range printed_lines {

		value := this.order.Lines[item]
		// log.Printf("%s:%f", item, value)
		if value > 0 || strings.Contains(item, "discount") {
			this.addLine(receiptLine(item, value))
			total += value
		}
	}

	// fmt.Printf("%s order total: %f, receipt total:%f \n", this.order.orderNo, this.order.total, total)
	this.curLine += 5
	this.pdf.SetLineWidth(1)
	this.pdf.SetLineType("solid")
	this.pdf.Line(8, this.curLine, 218.0, this.curLine)
	this.pdf.SetFont("Courier", "B", 7)
	if this.order.Tipe == "customer" {
		this.addLine(receiptLine("total_amount_"+this.order.Tipe, this.order.TotalCustomer))
	} else {
		this.addLine(receiptLine("total_amount_"+this.order.Tipe, this.order.TotalDriver))
	}
}

func (this *Receipt) printFooter() {

	this.pdf.SetFont("Courier", "", 7)
	this.curLine -= 8.0
	this.addLine(center("after_tax", 50)) //, {align:'center'} );

	this.curLine += 4

	this.pdf.SetLineWidth(1)
	this.pdf.SetLineType("dashed")
	this.pdf.Line(8, this.curLine, 218.0, this.curLine)

	this.curLine += 8
	this.pdf.SetFont("Courier", "B", 7)
	if this.company.Id == "ID01" {
		this.addLine(center(_t("No. Transaksi: ")+this.order.OrderNo, 49)) //, {align:'center'} );
	} else {
		this.addLine(center(_t("OrderNo: ")+this.order.OrderNo, 49)) //, {align:'center'} );
	}

	// r.pdf.font('Courier-Bold').text( this.order.service_type , {align:'center'} );
	this.pdf.SetFont("Courier", "", 7)
	order_date := this.order.OrderDate.Format(DATE_FORMAT)
	order_date = monthReplacer.Replace(order_date)
	order_date = dayReplacer.Replace(order_date)
	this.addLine(center(order_date, 50)) //, {align:"center"});
}

func (this *Receipt) renderCopyInfo(copyInfo string) {
	this.pdf.SetFont("Courier", "", 5)
	this.pdf.SetX(4)
	this.pdf.SetY(4)
	this.pdf.Text(fmt.Sprintf("%72s", copyInfo)) //,0,0, {align:"right"}) ;
}

func (this *Receipt) render(outStream io.Writer) []byte {
	this.printHeader()
	this.curLine += 16.0
	this.printItemLines()
	this.curLine += 16.0
	this.printFooter()
	this.renderCopyInfo(_t("second copy"))
	if outStream == nil {
		this.pdf.WritePdf(fmt.Sprintf("%s/%s", STORAGE_PREFIX_PATH, this.order.getReceiptPath(this.company.Id)))
	} else {
		this.pdf.WriteTo(outStream)
	}
	return this.pdf.GetBytesPdf()
}
