package pricing

import "testing"

func TestParseOnDemandUSDPrice(t *testing.T) {
	priceListItem := `{
		"terms": {
			"OnDemand": {
				"ABCDEF.JRTCKXETXF": {
					"offerTermCode": "JRTCKXETXF",
					"sku": "ABCDEF",
					"priceDimensions": {
						"ABCDEF.JRTCKXETXF.6YS6EN2CT7": {
							"pricePerUnit": {
								"USD": "0.1000000000"
							}
						}
					}
				}
			}
		}
	}`

	price, err := parseOnDemandUSDPrice(priceListItem)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if price != 0.1 {
		t.Fatalf("expected price 0.1, got %v", price)
	}
}

func TestParseOnDemandUSDPriceNoUSD(t *testing.T) {
	priceListItem := `{
		"terms": {
			"OnDemand": {
				"ABCDEF.JRTCKXETXF": {
					"priceDimensions": {
						"ABCDEF.JRTCKXETXF.6YS6EN2CT7": {
							"pricePerUnit": {
								"EUR": "0.0900000000"
							}
						}
					}
				}
			}
		}
	}`

	_, err := parseOnDemandUSDPrice(priceListItem)
	if err == nil {
		t.Fatal("expected error when USD price dimension is missing")
	}
}

func TestFindOnDemandUSDPriceSkipsInvalidPayloads(t *testing.T) {
	invalid := `{
		"terms": {
			"OnDemand": {
				"ABCDEF.JRTCKXETXF": {
					"priceDimensions": {
						"ABCDEF.JRTCKXETXF.6YS6EN2CT7": {
							"pricePerUnit": {
								"EUR": "0.0900000000"
							}
						}
					}
				}
			}
		}
	}`

	valid := `{
		"terms": {
			"OnDemand": {
				"ABCDEF.JRTCKXETXF": {
					"priceDimensions": {
						"ABCDEF.JRTCKXETXF.6YS6EN2CT7": {
							"pricePerUnit": {
								"USD": "0.2500000000"
							}
						}
					}
				}
			}
		}
	}`

	price, err := findOnDemandUSDPrice([]string{invalid, valid})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if price != 0.25 {
		t.Fatalf("expected price 0.25, got %v", price)
	}
}
