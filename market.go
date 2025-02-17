package steam

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	CurrencyUSD = "1"
	CurrencyGBP = "2"
	CurrencyEUR = "3"
	CurrencyCHF = "4"
	CurrencyRUB = "5"
	CurrencyPLN = "6"
	CurrencyBRL = "7"
	CurrencyJPY = "8"
	CurrencyNOK = "9"
	CurrencyIDR = "10"
	CurrencyMYR = "11"
	CurrencyPHP = "12"
	CurrencySGD = "13"
	CurrencyTHB = "14"
	CurrencyVND = "15"
	CurrencyKRW = "16"
	CurrencyTRY = "17"
	CurrencyUAH = "18"
	CurrencyMXN = "19"
	CurrencyCAD = "20"
	CurrencyAUD = "21"
	CurrencyNZD = "22"
	CurrencyCNY = "23"
	CurrencyINR = "24"
	CurrencyCLP = "25"
	CurrencyPEN = "26"
	CurrencyCOP = "27"
	CurrencyZAR = "28"
	CurrencyHKD = "29"
	CurrencyTWD = "30"
	CurrencySAR = "31"
	CurrencyAED = "32"
	CurrencyARS = "34"
	CurrencyILS = "35"
	CurrencyBYN = "36"
	CurrencyKZT = "37"
	CurrencyKWD = "38"
	CurrencyQAR = "39"
	CurrencyCRC = "40"
	CurrencyUYU = "41"
	CurrencyRMB = "9000"
)

type MarketItemPriceOverview struct {
	Success     bool   `json:"success"`
	LowestPrice string `json:"lowest_price"`
	MedianPrice string `json:"median_price"`
	Volume      string `json:"volume"`
}

type MarketItemPrice struct {
	Date  string
	Price float64
	Count string
}

type MarketItemResponse struct {
	Success     bool        `json:"success"`
	PricePrefix string      `json:"price_prefix"`
	PriceSuffix string      `json:"price_suffix"`
	Prices      interface{} `json:"prices"`
}

type MarketItemSearchResponse struct {
	Success    bool        `json:"success"`
	Start      int         `json:"start"`
	PageSize   int         `json:"pagesize"`
	TotalCount int         `json:"total_count"`
	SearchData interface{} `json:"searchdata"`
	Results    interface{} `json:"results"`
}

type MarketSearchItem struct {
	Name         string      `json:"name"`
	HashName     string      `json:"hash_name"`
	SellListings float64     `json:"sell_listings"`
	SellPrice    float64     `json:"sell_price"`
	SellPrice2   string      `json:"sell_price_text"` // The one user sees on search result list
	AppIcon      string      `json:"app_icon"`
	AppName      string      `json:"app_name"`
	AssetDesc    interface{} `json:"asset_description"`
	SalePrice    string      `json:"sale_price_text"` // The cheapest one available
}

type MarketSellResponse struct {
	Success                    bool   `json:"success"`
	RequiresConfirmation       uint32 `json:"requires_confirmation"`
	MobileConfirmationRequired bool   `json:"needs_mobile_confirmation"`
	EmailConfirmationRequired  bool   `json:"needs_email_confirmation"`
	EmailDomain                string `json:"email_domain"`
}

type MarketBuyOrderResponse struct {
	ErrCode int    `json:"success"`
	ErrMsg  string `json:"message"` // Set if ErrCode != 1
	OrderID uint64 `json:"buy_orderid,string"`
}

var (
	ErrCannotLoadPrices     = errors.New("unable to load prices at this time")
	ErrInvalidPriceResponse = errors.New("invalid market pricehistory response")
)

func (session *Session) GetMarketItemPriceHistory(appID uint64, marketHashName string) ([]*MarketItemPrice, error) {
	resp, err := session.client.Get("https://steamcommunity.com/market/pricehistory/?" + url.Values{
		"appid":            {strconv.FormatUint(appID, 10)},
		"market_hash_name": {marketHashName},
	}.Encode())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %d", resp.StatusCode)
	}

	response := MarketItemResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if !response.Success {
		return nil, ErrCannotLoadPrices
	}

	var prices []interface{}
	var ok bool
	if prices, ok = response.Prices.([]interface{}); !ok {
		return nil, ErrCannotLoadPrices
	}

	items := []*MarketItemPrice{}
	for _, v := range prices {
		if v, ok := v.([]interface{}); ok {
			item := &MarketItemPrice{}
			for _, val := range v {
				switch val := val.(type) {
				case string:
					if len(item.Date) != 0 {
						item.Count = val
					} else {
						item.Date = val
					}
				case float64:
					item.Price = val
				}
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (session *Session) GetMarketItemPriceOverview(appID uint64, country, currencyID, marketHashName string) (*MarketItemPriceOverview, error) {
	resp, err := session.client.Get("https://steamcommunity.com/market/priceoverview/?" + url.Values{
		"appid":            {strconv.FormatUint(appID, 10)},
		"country":          {country},
		"currencyID":       {currencyID},
		"market_hash_name": {marketHashName},
	}.Encode())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %d", resp.StatusCode)
	}

	overview := &MarketItemPriceOverview{}
	if err = json.NewDecoder(resp.Body).Decode(overview); err != nil {
		return nil, err
	}

	return overview, nil
}

func (session *Session) GetMarketItemSearch(appID uint64, searchQuery string, offset int, count int) (*MarketItemSearchResponse, []*MarketSearchItem, error) {
	resp, err := session.client.Get("https://steamcommunity.com/market/search/render/?norender=1&" + url.Values{
		"appid":  {strconv.FormatUint(appID, 10)},
		"query":  {searchQuery},
		"offset": {strconv.Itoa(offset)},
		"count":  {strconv.Itoa(count)},
	}.Encode())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("http error: %d", resp.StatusCode)
	}

	response := &MarketItemSearchResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, nil, err
	}

	if !response.Success {
		return nil, nil, ErrCannotLoadPrices
	}

	var results []interface{}
	var ok bool
	if results, ok = response.Results.([]interface{}); !ok {
		return nil, nil, ErrCannotLoadPrices
	}

	items := []*MarketSearchItem{}
	for _, v := range results {
		if v, ok := v.(map[string]interface{}); ok {
			item := &MarketSearchItem{}

			item.Name = v["name"].(string)
			item.HashName = v["hash_name"].(string)
			item.SellListings = v["sell_listings"].(float64)
			item.SellPrice = v["sell_price"].(float64)
			item.SellPrice2 = v["sell_price_text"].(string)
			item.AppIcon = v["app_icon"].(string)
			item.AppName = v["app_name"].(string)
			item.AssetDesc = v["asset_description"]
			item.SalePrice = v["sale_price_text"].(string)

			items = append(items, item)
		}
	}

	return response, items, nil
}

func (session *Session) SellItem(item *InventoryItem, amount, price uint64) (*MarketSellResponse, error) {
	resp, err := session.client.PostForm("https://steamcommunity.com/market/sellitem/", url.Values{
		"amount":    {strconv.FormatUint(amount, 10)},
		"appid":     {strconv.FormatUint(uint64(item.AppID), 10)},
		"assetid":   {strconv.FormatUint(item.AssetID, 10)},
		"contextid": {strconv.FormatUint(item.ContextID, 10)},
		"price":     {strconv.FormatUint(price, 10)},
		"sessionid": {session.sessionID},
	})
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %d", resp.StatusCode)
	}

	response := &MarketSellResponse{}
	if err = json.NewDecoder(resp.Body).Decode(response); err != nil {
		return nil, err
	}

	return response, nil
}

func (session *Session) PlaceBuyOrder(appid uint64, priceTotal float64, quantity uint64, currencyID, marketHashName string) (*MarketBuyOrderResponse, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		"https://steamcommunity.com/market/createbuyorder/",
		strings.NewReader(url.Values{
			"appid":            {strconv.FormatUint(appid, 10)},
			"currency":         {currencyID},
			"market_hash_name": {marketHashName},
			"price_total":      {strconv.FormatUint(uint64(priceTotal*100), 10)},
			"quantity":         {strconv.FormatUint(quantity, 10)},
			"sessionid":        {session.sessionID},
		}.Encode()),
	)
	if err != nil {
		return nil, err
	}

	var referer string
	referer = strings.Replace(marketHashName, " ", "%20", -1)
	referer = strings.Replace(referer, "#", "%23", -1)

	req.Header.Add(
		"Referer",
		fmt.Sprintf("https://steamcommunity.com/market/listings/%d/%s", appid, referer),
	)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := session.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	response := &MarketBuyOrderResponse{}
	if err = json.NewDecoder(resp.Body).Decode(response); err != nil {
		return nil, err
	}

	return response, nil
}

func (session *Session) CancelBuyOrder(orderid uint64) error {
	req, err := http.NewRequest(
		http.MethodPost,
		"https://steamcommunity.com/market/cancelbuyorder/",
		strings.NewReader(url.Values{
			"sessionid":   {session.sessionID},
			"buy_orderid": {strconv.FormatUint(orderid, 10)},
		}.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Add("Referer", "https://steamcommunity.com/market")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := session.client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cannot cancel %d: %d", orderid, resp.StatusCode)
	}

	return nil
}
