package tax_receipt

type Company struct {
	Id      string
	Name    string
	NPWP    string
	Address []string
}

var companies = map[string]Company{
	"ID01": Company{"ID01", "PT ABC.", "98.765.432.1-064.000", []string{"Gd. B Lt 6&7", "Jl. Dimana ya", "Jakarta Selatan, DKI Jakarta"}},
	"ID04": Company{"ID04", "PT XYZ", "12.345.678.9-411.000", []string{"Jl. Jalan II No. 2", "Kelurahan, Kecamatan", "Jakarta, DKI Jakarta"}},
	"ID13": Company{"ID13", "PT 123", "12.345.678.0-064.000", []string{"Jl. Street II No. 2", "Kelurahan, Kecamatan", "Jakarta , DKI Jakarta"}},
}

func getCompany(companyId string) Company {
	return companies[companyId]
}
