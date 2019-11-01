package migrations

import (
	"encoding/json"
	"github.com/OpenBazaar/jsonpb"
	"github.com/OpenBazaar/openbazaar-go/ipfs"
	"github.com/OpenBazaar/openbazaar-go/pb"
	"github.com/golang/protobuf/proto"
	coremock "github.com/ipfs/go-ipfs/core/mock"
	crypto "gx/ipfs/QmTW4SdgBWq9GjsBsHeUx8WuGxzhgzAf88UMH2w62PC8yK/go-libp2p-crypto"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"strconv"
	"strings"
)

type Migration027 struct{}

type (
	Migration027ListingThumbnail struct {
		Tiny   string `json:"tiny"`
		Small  string `json:"small"`
		Medium string `json:"medium"`
	}

	Migration027V5CurrencyValue struct {
		Amount   *big.Int                         `json:"amount"`
		Currency Migration027V5CurrencyDefinition `json:"currency"`
	}

	Migration027V5CurrencyCode string

	Migration027V5CurrencyDefinition struct {
		Code         Migration027V5CurrencyCode `json:"code"`
		Divisibility uint                       `json:"divisibility"`
	}

	Migration027V5ListingIndexData struct {
		Hash               string                       `json:"hash"`
		Slug               string                       `json:"slug"`
		Title              string                       `json:"title"`
		Categories         []string                     `json:"categories"`
		NSFW               bool                         `json:"nsfw"`
		ContractType       string                       `json:"contractType"`
		Description        string                       `json:"description"`
		Thumbnail          Migration027ListingThumbnail `json:"thumbnail"`
		Price              *Migration027V5CurrencyValue `json:"price"`
		Modifier           float32                      `json:"modifier"`
		ShipsTo            []string                     `json:"shipsTo"`
		FreeShipping       []string                     `json:"freeShipping"`
		Language           string                       `json:"language"`
		AverageRating      float32                      `json:"averageRating"`
		RatingCount        uint32                       `json:"ratingCount"`
		ModeratorIDs       []string                     `json:"moderators"`
		AcceptedCurrencies []string                     `json:"acceptedCurrencies"`
		CryptoCurrencyCode string                       `json:"coinType"`
	}

	Migration027V4price struct {
		CurrencyCode string  `json:"currencyCode"`
		Amount       uint    `json:"amount"`
		Modifier     float32 `json:"modifier"`
	}

	Migration027V4ListingIndexData struct {
		Hash               string                       `json:"hash"`
		Slug               string                       `json:"slug"`
		Title              string                       `json:"title"`
		Categories         []string                     `json:"categories"`
		NSFW               bool                         `json:"nsfw"`
		ContractType       string                       `json:"contractType"`
		Description        string                       `json:"description"`
		Thumbnail          Migration027ListingThumbnail `json:"thumbnail"`
		Price              Migration027V4price          `json:"price"`
		ShipsTo            []string                     `json:"shipsTo"`
		FreeShipping       []string                     `json:"freeShipping"`
		Language           string                       `json:"language"`
		AverageRating      float32                      `json:"averageRating"`
		RatingCount        uint32                       `json:"ratingCount"`
		ModeratorIDs       []string                     `json:"moderators"`
		AcceptedCurrencies []string                     `json:"acceptedCurrencies"`
		CryptoCurrencyCode string                       `json:"coinType"`
	}
)

var divisibilityMap = map[string]uint{
	"BTC": 8,
	"BCH": 8,
	"LTC": 8,
	"ZEC": 8,
}

func parseV5intoV4(v5 Migration027V5ListingIndexData) Migration027V4ListingIndexData {
	return Migration027V4ListingIndexData{
		Hash:         v5.Hash,
		Slug:         v5.Slug,
		Title:        v5.Title,
		Categories:   v5.Categories,
		NSFW:         v5.NSFW,
		ContractType: v5.ContractType,
		Description:  v5.Description,
		Thumbnail:    v5.Thumbnail,
		Price: Migration027V4price{
			CurrencyCode: string(v5.Price.Currency.Code),
			Amount:       uint(v5.Price.Amount.Uint64()),
			Modifier:     v5.Modifier,
		},
		ShipsTo:            v5.ShipsTo,
		FreeShipping:       v5.FreeShipping,
		Language:           v5.Language,
		AverageRating:      v5.AverageRating,
		RatingCount:        v5.RatingCount,
		ModeratorIDs:       v5.ModeratorIDs,
		AcceptedCurrencies: v5.AcceptedCurrencies,
		CryptoCurrencyCode: v5.CryptoCurrencyCode,
	}
}

func parseV4intoV5(v4 Migration027V4ListingIndexData) Migration027V5ListingIndexData {
	var priceValue *Migration027V5CurrencyValue
	divisibility, ok := divisibilityMap[strings.ToUpper(v4.Price.CurrencyCode)]
	if !ok {
		divisibility = 2
	}

	priceValue = &Migration027V5CurrencyValue{
		Amount: new(big.Int).SetInt64(int64(v4.Price.Amount)),
		Currency: Migration027V5CurrencyDefinition{
			Code:         Migration027V5CurrencyCode(v4.Price.CurrencyCode),
			Divisibility: divisibility,
		},
	}
	return Migration027V5ListingIndexData{
		Hash:               v4.Hash,
		Slug:               v4.Slug,
		Title:              v4.Title,
		Categories:         v4.Categories,
		NSFW:               v4.NSFW,
		ContractType:       v4.ContractType,
		Description:        v4.Description,
		Thumbnail:          v4.Thumbnail,
		Modifier:           v4.Price.Modifier,
		Price:              priceValue,
		ShipsTo:            v4.ShipsTo,
		FreeShipping:       v4.FreeShipping,
		Language:           v4.Language,
		AverageRating:      v4.AverageRating,
		RatingCount:        v4.RatingCount,
		ModeratorIDs:       v4.ModeratorIDs,
		AcceptedCurrencies: v4.AcceptedCurrencies,
		CryptoCurrencyCode: v4.CryptoCurrencyCode,
	}
}

func (Migration027) Up(repoPath, databasePassword string, testnetEnabled bool) error {
	listingsFilePath := path.Join(repoPath, "root", "listings.json")

	// Non-vendors might not have an listing.json and we don't want to error here if that's the case
	indexExists := true
	if _, err := os.Stat(listingsFilePath); os.IsNotExist(err) {
		indexExists = false
	}

	if indexExists {
		// Setup signing capabilities
		identityKey, err := migration027_GetIdentityKey(repoPath, databasePassword, testnetEnabled)
		if err != nil {
			return err
		}

		sk, err := crypto.UnmarshalPrivateKey(identityKey)
		if err != nil {
			return err
		}

		nd, err := coremock.NewMockNode()
		if err != nil {
			return err
		}

		listingHashMap := make(map[string]string)

		m := jsonpb.Marshaler{
			Indent: "    ",
		}

		var oldListingIndex []Migration027V4ListingIndexData
		listingsJSON, err := ioutil.ReadFile(listingsFilePath)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(listingsJSON, &oldListingIndex); err != nil {
			return err
		}

		newListingIndex := make([]Migration027V5ListingIndexData, len(oldListingIndex))
		for i, listing := range oldListingIndex {
			newListingIndex[i] = parseV4intoV5(listing)
		}

		for _, listing := range newListingIndex {
			listingPath := path.Join(repoPath, "root", "listings", listing.Slug+".json")
			listingBytes, err := ioutil.ReadFile(listingPath)
			if err != nil {
				return err
			}
			var signedListingJSON map[string]interface{}
			if err = json.Unmarshal(listingBytes, &signedListingJSON); err != nil {
				return err
			}

			listingJSON := signedListingJSON["listing"]
			listing := listingJSON.(map[string]interface{})

			metadataJSON := listing["metadata"]
			metadata := metadataJSON.(map[string]interface{})
			itemJSON := listing["item"]
			item := itemJSON.(map[string]interface{})

			var (
				skus            []interface{}
				shippingOptions []interface{}
				coupons         []interface{}
			)

			skusJSON := item["skus"]
			if skusJSON != nil {
				skus = skusJSON.([]interface{})
			}
			shippingOptionsJSON := listing["shippingOptions"]
			if shippingOptionsJSON != nil {
				shippingOptions = shippingOptionsJSON.([]interface{})
			}
			couponsJSON := listing["coupons"]
			if couponsJSON != nil {
				coupons = couponsJSON.([]interface{})
			}

			pricingCurrencyJSON := metadata["pricingCurrency"]
			pricingCurrency := pricingCurrencyJSON.(string)

			divisibility, ok := divisibilityMap[strings.ToUpper(pricingCurrency)]
			if !ok {
				divisibility = 2
			}

			item["priceCurrency"] = struct {
				Code         string `json:"code"`
				Divisibility uint32 `json:"divisibility"`
			}{
				Code:         pricingCurrency,
				Divisibility: uint32(divisibility),
			}

			delete(metadata, "pricingCurrency")

			var modifier float64
			modifierJSON := metadata["priceModifier"]
			if modifierJSON != nil {
				modifier = modifierJSON.(float64)
			}

			item["priceModifier"] = modifier

			delete(metadata, "priceModifier")

			priceJSON := item["price"]
			price := priceJSON.(float64)

			item["bigPrice"] = strconv.Itoa(int(price))

			delete(item, "price")

			var coinType string
			coinTypeJSON := metadata["coinType"]
			if coinTypeJSON != nil {
				coinType = coinTypeJSON.(string)
			}

			metadata["cryptoCurrencyCode"] = coinType

			delete(metadata, "coinType")

			var coinDivisibility float64
			coinDivisibilityJSON := metadata["coinDivisibility"]
			if coinDivisibilityJSON != nil {
				coinDivisibility = coinDivisibilityJSON.(float64)
			}

			metadata["cryptoDivisibility"] = uint32(coinDivisibility)

			delete(metadata, "coinDivisibility")

			for _, skuJSON := range skus {
				sku := skuJSON.(map[string]interface{})

				quantityJSON, ok := sku["quantity"]
				if ok {
					quantity := quantityJSON.(float64)

					sku["bigQuantity"] = strconv.Itoa(int(quantity))

					delete(sku, "quantity")
				}

				surchargeJSON, ok := sku["surcharge"]
				if ok {
					surcharge := surchargeJSON.(float64)

					sku["bigSurcharge"] = strconv.Itoa(int(surcharge))

					delete(sku, "surcharge")
				}
			}

			for i, shippingOptionJSON := range shippingOptions {
				so := shippingOptionJSON.(map[string]interface{})
				var services []interface{}
				servicesJSON := so["services"]
				if servicesJSON != nil {
					services = servicesJSON.([]interface{})
				}

				for x, serviceJSON := range services {
					service := serviceJSON.(map[string]interface{})

					priceJSON := service["price"]
					price := priceJSON.(float64)

					service["bigPrice"] = strconv.Itoa(int(price))

					delete(service, "price")

					additionalItemPriceJSON, ok := service["additionalItemPrice"]
					if ok {
						additionalItemPrice := additionalItemPriceJSON.(float64)

						service["bigAdditionalItemPrice"] = strconv.Itoa(int(additionalItemPrice))

						delete(service, "additionalItemPrice")
					}

					services[x] = service
				}

				so["services"] = services
				shippingOptions[i] = so
			}

			for _, couponJSON := range coupons {
				coupon := couponJSON.(map[string]interface{})

				priceDiscountJSON, ok := coupon["priceDiscount"]
				if ok {
					priceDiscount := priceDiscountJSON.(float64)

					coupon["bigPriceDiscount"] = strconv.Itoa(int(priceDiscount))

					delete(coupon, "priceDiscount")
				}
			}

			out, err := json.MarshalIndent(signedListingJSON, "", "    ")
			if err != nil {
				return err
			}

			sl := new(pb.SignedListing)
			if err := jsonpb.UnmarshalString(string(out), sl); err != nil {
				return err
			}

			ser, err := proto.Marshal(sl.Listing)
			if err != nil {
				return err
			}

			sig, err := sk.Sign(ser)
			if err != nil {
				return err
			}

			sl.Signature = sig

			signedOut, err := m.MarshalToString(sl)
			if err != nil {
				return err
			}

			err = ioutil.WriteFile(listingPath, []byte(signedOut), os.ModePerm)
			if err != nil {
				return err
			}

			hash, err := ipfs.GetHashOfFile(nd, listingPath)
			if err != nil {
				return err
			}

			listingHashMap[sl.Listing.Slug] = hash
		}

		for i, listing := range newListingIndex {
			newListingIndex[i].Hash = listingHashMap[listing.Slug]
		}

		migratedJSON, err := json.MarshalIndent(&newListingIndex, "", "    ")
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(listingsFilePath, migratedJSON, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return writeRepoVer(repoPath, 28)
}

func (Migration027) Down(repoPath, databasePassword string, testnetEnabled bool) error {
	listingsFilePath := path.Join(repoPath, "root", "listings.json")

	// Non-vendors might not have an listing.json and we don't want to error here if that's the case
	indexExists := true
	if _, err := os.Stat(listingsFilePath); os.IsNotExist(err) {
		indexExists = false
	}

	if indexExists {
		// Setup signing capabilities
		identityKey, err := migration027_GetIdentityKey(repoPath, databasePassword, testnetEnabled)
		if err != nil {
			return err
		}

		sk, err := crypto.UnmarshalPrivateKey(identityKey)
		if err != nil {
			return err
		}

		nd, err := coremock.NewMockNode()
		if err != nil {
			return err
		}

		listingHashMap := make(map[string]string)

		m := jsonpb.Marshaler{
			Indent: "    ",
		}

		var oldListingIndex []Migration027V5ListingIndexData
		listingsJSON, err := ioutil.ReadFile(listingsFilePath)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(listingsJSON, &oldListingIndex); err != nil {
			return err
		}

		newListingIndex := make([]Migration027V4ListingIndexData, len(oldListingIndex))
		for i, listing := range oldListingIndex {
			newListingIndex[i] = parseV5intoV4(listing)
		}

		for _, listing := range newListingIndex {
			listingPath := path.Join(repoPath, "root", "listings", listing.Slug+".json")
			listingBytes, err := ioutil.ReadFile(listingPath)
			if err != nil {
				return err
			}
			var signedListingJSON map[string]interface{}
			if err = json.Unmarshal(listingBytes, &signedListingJSON); err != nil {
				return err
			}

			listingJSON := signedListingJSON["listing"]
			listing := listingJSON.(map[string]interface{})

			metadataJSON := listing["metadata"]
			metadata := metadataJSON.(map[string]interface{})
			itemJSON := listing["item"]
			item := itemJSON.(map[string]interface{})

			var (
				skus            []interface{}
				shippingOptions []interface{}
				coupons         []interface{}
			)

			skusJSON := item["skus"]
			if skusJSON != nil {
				skus = skusJSON.([]interface{})
			}
			shippingOptionsJSON := listing["shippingOptions"]
			if shippingOptionsJSON != nil {
				shippingOptions = shippingOptionsJSON.([]interface{})
			}
			couponsJSON := listing["coupons"]
			if couponsJSON != nil {
				coupons = couponsJSON.([]interface{})
			}

			pricingCurrencyJSON := item["priceCurrency"]
			pricingCurrency := pricingCurrencyJSON.(map[string]interface{})

			priceCurrencyCodeJSON := pricingCurrency["code"]
			priceCurrencyCode := priceCurrencyCodeJSON.(string)

			metadata["pricingCurrency"] = priceCurrencyCode

			delete(item, "priceCurrency")

			var modifier float64
			modifierJSON := item["priceModifier"]
			if modifierJSON != nil {
				modifier = modifierJSON.(float64)
			}

			metadata["priceModifier"] = modifier

			delete(item, "priceModifier")

			priceJSON := item["bigPrice"]
			price := priceJSON.(string)

			p, ok := new(big.Int).SetString(price, 10)
			if ok {
				item["price"] = p.Uint64()
			}
			delete(item, "bigPrice")

			var coinType string
			coinTypeJSON := metadata["cryptoCurrencyCode"]
			if coinTypeJSON != nil {
				coinType = coinTypeJSON.(string)
			}

			metadata["coinType"] = coinType

			delete(metadata, "cryptoCurrencyCode")

			var coinDivisibility float64
			coinDivisibilityJSON := metadata["cryptoDivisibility"]
			if coinDivisibilityJSON != nil {
				coinDivisibility = coinDivisibilityJSON.(float64)
			}

			metadata["coinDivisibility"] = uint32(coinDivisibility)

			delete(metadata, "cryptoDivisibility")

			for _, skuJSON := range skus {
				sku := skuJSON.(map[string]interface{})
				quantityJSON, ok := sku["bigQuantity"]
				if ok {
					quantity := quantityJSON.(string)

					p, ok := new(big.Int).SetString(quantity, 10)
					if ok {
						sku["quantity"] = p.Uint64()
					}

					delete(sku, "bigQuantity")
				}

				surchargeJSON, ok := sku["bigSurcharge"]
				if ok {
					surcharge := surchargeJSON.(string)

					s, ok := new(big.Int).SetString(surcharge, 10)
					if ok {
						sku["surcharge"] = s.Uint64()
					}

					delete(sku, "bigSurcharge")
				}
			}

			for i, shippingOptionJSON := range shippingOptions {
				so := shippingOptionJSON.(map[string]interface{})
				var services []interface{}
				servicesJSON := so["services"]
				if servicesJSON != nil {
					services = servicesJSON.([]interface{})
				}

				for x, serviceJSON := range services {
					service := serviceJSON.(map[string]interface{})

					priceJSON := service["bigPrice"]
					price := priceJSON.(string)

					p, ok := new(big.Int).SetString(price, 10)
					if ok {
						service["price"] = p.Uint64()
					}

					delete(service, "bigPrice")

					additionalItemPriceJSON, ok := service["bigAdditionalItemPrice"]
					if ok {
						additionalItemPrice := additionalItemPriceJSON.(string)

						a, ok := new(big.Int).SetString(additionalItemPrice, 10)
						if ok {
							service["additionalItemPrice"] = a.Uint64()
						}

						delete(service, "bigAdditionalItemPrice")
					}

					services[x] = service
				}

				shippingOptions[i] = so
			}

			for _, couponJSON := range coupons {
				coupon := couponJSON.(map[string]interface{})

				priceDiscountJSON, ok := coupon["bigPriceDiscount"]
				if ok {
					priceDiscount := priceDiscountJSON.(string)

					a, ok := new(big.Int).SetString(priceDiscount, 10)
					if ok {
						coupon["priceDiscount"] = a.Uint64()
					}

					delete(coupon, "bigPriceDiscount")
				}
			}

			out, err := json.MarshalIndent(signedListingJSON, "", "    ")
			if err != nil {
				return err
			}

			sl := new(pb.SignedListing)
			if err := jsonpb.UnmarshalString(string(out), sl); err != nil {
				return err
			}

			ser, err := proto.Marshal(sl.Listing)
			if err != nil {
				return err
			}

			sig, err := sk.Sign(ser)
			if err != nil {
				return err
			}

			sl.Signature = sig

			signedOut, err := m.MarshalToString(sl)
			if err != nil {
				return err
			}

			err = ioutil.WriteFile(listingPath, []byte(signedOut), os.ModePerm)
			if err != nil {
				return err
			}

			hash, err := ipfs.GetHashOfFile(nd, listingPath)
			if err != nil {
				return err
			}

			listingHashMap[sl.Listing.Slug] = hash
		}

		for i, listing := range newListingIndex {
			newListingIndex[i].Hash = listingHashMap[listing.Slug]
		}

		migratedJSON, err := json.MarshalIndent(&newListingIndex, "", "    ")
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(listingsFilePath, migratedJSON, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return writeRepoVer(repoPath, 27)
}

func migration027_GetIdentityKey(repoPath, databasePassword string, testnetEnabled bool) ([]byte, error) {
	db, err := OpenDB(repoPath, databasePassword, testnetEnabled)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var identityKey []byte
	err = db.
		QueryRow("select value from config where key=?", "identityKey").
		Scan(&identityKey)
	if err != nil {
		return nil, err
	}
	return identityKey, nil
}
