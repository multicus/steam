package steam

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"strconv"
)

// Due to the JSON being string, etc... we cannot re-use EconItem
// Also, "assetid" is included as "id" not as assetid.
type InventoryItem struct {
	AssetID        uint64 `json:"id,string,omitempty"`
	InstanceID     uint64 `json:"instanceid,string,omitempty"`
	ClassID        uint64 `json:"classid,string,omitempty"`
	AppID          uint32 `json:"appid"`     // This!
	ContextID      uint64 `json:"contextid"` // Ditto
	Name           string `json:"name"`
	MarketName     string `json:"market_name"`
	MarketHashName string `json:"market_hash_name"`
}

type InventoryContext struct {
	ID         uint64 `json:"id,string"` /* Apparently context id needs at least 64 bits...  */
	AssetCount uint32 `json:"asset_count"`
	Name       string `json:"name"`
}

type InventoryAppStats struct {
	AppID            uint32                       `json:"appid"`
	Name             string                       `json:"name"`
	AssetCount       uint32                       `json:"asset_count"`
	Icon             string                       `json:"icon"`
	Link             string                       `json:"link"`
	InventoryLogo    string                       `json:"inventory_logo"`
	TradePermissions string                       `json:"trade_permissions"`
	Contexts         map[string]*InventoryContext `json:"rgContexts"`
}

var (
	InventoryContextRegexp = regexp.MustCompile("var g_rgAppContextData = (.*?);")
	ErrCannotLoadInventory = errors.New("unable to load inventory at this time")
)

func (community *Community) parseInventory(sid SteamID, appID uint32, contextID uint64, start uint32, tradableOnly bool, items *[]*InventoryItem) (uint32, error) {
	params := url.Values{
		"start": {strconv.FormatUint(uint64(start), 10)},
	}
	if tradableOnly {
		params.Set("trading", "1")
	}

	resp, err := community.client.Get(fmt.Sprintf("https://steamcommunity.com/profiles/%d/inventory/json/%d/%d/?", sid, appID, contextID) + params.Encode())
	if err != nil {
		return 0, err
	}

	type DescItem struct {
		Name           string `json:"name"`
		MarketName     string `json:"market_name"`
		MarketHashName string `json:"market_hash_name"`
	}

	type Response struct {
		Success      bool                      `json:"success"`
		MoreStart    interface{}               `json:"more_start"` // This can be a bool or a number...
		Inventory    map[string]*InventoryItem `json:"rgInventory"`
		Descriptions map[string]*DescItem      `json:"rgDescriptions"`
		/* Missing: rgCurrency  */
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, err
	}

	if !response.Success {
		return 0, ErrCannotLoadInventory
	}

	// Morph response.Inventory into an array of items.
	// This is due to Steam returning the items in the following format:
	//	rgInventory: {
	//		"54xxx": {
	//			"id": "54xxx"
	//			...
	//		}
	//	}
	for _, value := range response.Inventory {
		desc, ok := response.Descriptions[strconv.FormatUint(value.ClassID, 10)+"_"+strconv.FormatUint(value.InstanceID, 10)]
		if ok {
			value.Name = desc.Name
			value.MarketName = desc.Name
			value.MarketHashName = desc.MarketHashName
		}

		*items = append(*items, value)
	}

	switch response.MoreStart.(type) {
	case int, uint:
		return uint32(response.MoreStart.(int)), nil
	case bool:
		break
	default:
		return 0, fmt.Errorf("parseInventory(): Please implement case for type %v", response.MoreStart)
	}

	return 0, nil
}

func (community *Community) GetInventory(sid SteamID, appID uint32, contextID uint64, tradableOnly bool) ([]*InventoryItem, error) {
	items := []*InventoryItem{}
	more := uint32(0)

	for {
		next, err := community.parseInventory(sid, appID, contextID, more, tradableOnly, &items)
		if err != nil {
			return nil, err
		}

		if next == 0 {
			break
		}

		more = next
	}

	return items, nil
}

func (community *Community) GetInventoryAppStats(sid SteamID) (map[string]InventoryAppStats, error) {
	resp, err := community.client.Get("https://steamcommunity.com/profiles/" + sid.ToString() + "/inventory")
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	m := InventoryContextRegexp.FindSubmatch(body)
	if m == nil || len(m) != 2 {
		return nil, err
	}

	inven := map[string]InventoryAppStats{}
	if err = json.Unmarshal(m[1], &inven); err != nil {
		return nil, err
	}

	return inven, nil
}
