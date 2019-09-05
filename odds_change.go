package uof

import (
	"encoding/xml"
	"strings"
)

// OddsChange messages are sent whenever Betradar has new odds for some markets
// for a match. Odds changes can include a subset of all markets; if so, markets
// not reported remain unchanged. All outcomes possible within a market are
// reported.
// Reference: https://docs.betradar.com/display/BD/UOF+-+Odds+change
type OddsChange struct {
	EventURN URN `xml:"event_id,attr" json:"eventURN"`
	// Specifies which producer generated these odds. At any given point in time
	// there should only be one product generating odds for a particular event.
	Product                  Producer                  `xml:"product,attr" json:"product"`
	Timestamp                int64                     `xml:"timestamp,attr" json:"timestamp"`
	Odds                     *Odds                     `xml:"odds,omitempty" json:"odds,omitempty"`
	SportEventStatus         *SportEventStatus         `xml:"sport_event_status,omitempty" json:"sportEventStatus,omitempty"`
	OddsChangeReason         *uint8                    `xml:"odds_change_reason,attr,omitempty" json:"oddsChangeReason,omitempty"` // May be one of 1
	OddsGenerationProperties *OddsGenerationProperties `xml:"odds_generation_properties,omitempty" json:"oddsGenerationProperties,omitempty"`
	RequestID                *int64                    `xml:"request_id,attr,omitempty" json:"requestID,omitempty"`
}
type OddsGenerationProperties struct {
	ExpectedTotals    *float64 `xml:"expected_totals,attr,omitempty" json:"expectedTotals,omitempty"`
	ExpectedSupremacy *float64 `xml:"expected_supremacy,attr,omitempty" json:"expectedSupremacy,omitempty"`
}

type Odds struct {
	Markets []Market `xml:"market,omitempty" json:"market,omitempty"`
	// values in range 0-6   /v1/descriptions/betting_status.xml
	BettingStatus *int `xml:"betting_status,attr,omitempty" json:"bettingStatus,omitempty"`
	// values in range 0-87  /v1/descriptions/betstop_reasons.xml
	BetstopReason *int `xml:"betstop_reason,attr,omitempty" json:"betstopReason,omitempty"`
}

// Market describes the odds updates for a particular market.
// Betradar Unified Odds utilizes markets and market lines. Each market is a bet
// type identified with a unique ID and within a market, multiple different lines
// are often provided. Each of these lines is uniquely identified by additional
// specifiers (e.g. Total Goals 2.5 is the same market as Total Goals 1.5, but it
// is two different market lines. The market ID for both are the same, but the
// first one has a specifier ((goals=2.5)) and the other one has a specifier
// ((goals=1.5)) that uniquely identifies them).
// LineID is hash of specifier field used to uniquely identify lines in one market.
// One market line is uniquely identified by market id and line id.
type Market struct {
	ID             int               `xml:"id,attr" json:"id"`
	LineID         int               `json:"lineID"`
	Specifiers     map[string]string `json:"sepcifiers,omitempty"`
	Status         MarketStatus      `xml:"status,attr,omitempty" json:"status,omitempty"`
	CashoutStatus  *CashoutStatus    `xml:"cashout_status,attr,omitempty" json:"cashoutStatus,omitempty"`
	Favourite      *bool             `xml:"favourite,attr,omitempty" json:"favourite,omitempty"`
	Outcomes       []Outcome         `xml:"outcome,omitempty" json:"outcome,omitempty"`
	MarketMetadata *MarketMetadata   `xml:"market_metadata,omitempty" json:"marketMetadata,omitempty"`
}
type MarketMetadata struct {
	NextBetstop *int64 `xml:"next_betstop,attr,omitempty" json:"nextBetstop,omitempty"`
}

type Outcome struct {
	URN           URN      `xml:"id,attr" json:"id"`
	Odds          *float64 `xml:"odds,attr,omitempty" json:"odds,omitempty"`
	Probabilities *float64 `xml:"probabilities,attr,omitempty" json:"probabilities,omitempty"`
	Active        *bool    `xml:"active,attr,omitempty" json:"active,omitempty"`
	Team          *Team    `xml:"team,attr,omitempty" json:"team,omitempty"`
}

// Custom unmarshaling reasons:
//  * To cover the case that: 'The default value is active if status is not present.'
//  * To convert Specifiers and ExtendedSpecifiers fileds which are
//    lists of key value attributes encoded in string to the map.
//  * To calculate LineID; market line is uniquely identified by both
//    market id and line id
func (t *Market) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type T Market
	var overlay struct {
		*T
		Status             *int8  `xml:"status,attr,omitempty"`
		Specifiers         string `xml:"specifiers,attr,omitempty" json:"specifiers,omitempty"`
		ExtendedSpecifiers string `xml:"extended_specifiers,attr,omitempty" json:"extendedSpecifiers,omitempty"`
	}
	overlay.T = (*T)(t)
	if err := d.DecodeElement(&overlay, &start); err != nil {
		return err
	}
	overlay.T.Status = MarketStatusActive // default
	if overlay.Status != nil {
		overlay.T.Status = MarketStatus(*overlay.Status)
	}
	overlay.T.Specifiers = toSpecifiers(overlay.Specifiers, overlay.ExtendedSpecifiers)
	overlay.T.LineID = toLineID(overlay.Specifiers)
	return nil
}

func (o *OddsChange) EventID() int {
	return o.EventURN.ID()
}

// func (o OddsChange) Scope() uint8 {
// 	//1 =	LiveOdds producer, 3 = Betradar Ctrl producer, 4 = BetPal producer, 5 = Premium Cricket producer
// 	if o.Product == 3 {
// 		return uof.Prematch
// 	}
// 	return uof.Live
// }

func (m Market) VariantSpecifier() string {
	for k, v := range m.Specifiers {
		if k == "variant" {
			return v
		}
	}
	return ""
}

// func (m Market) VariantID() *uint32 {
// 	if m.SRSpecifiers == nil {
// 		return nil
// 	}
// 	id := toVariantID(m.VariantSpecifier())
// 	return &id
// }

func toSpecifiers(specifiers, extendedSpecifiers string) map[string]string {
	allSpecifiers := specifiers
	if extendedSpecifiers != "" {
		allSpecifiers = allSpecifiers + "|" + extendedSpecifiers
	}
	if len(allSpecifiers) < 2 {
		return nil
	}
	sm := make(map[string]string)
	for _, s := range strings.Split(allSpecifiers, "|") {
		if p := strings.Split(s, "="); len(p) == 2 {
			k := p[0]
			v := p[1]
			if k == "player" {
				v = strings.TrimPrefix(v, srPlayer)
			}
			sm[k] = v
		}
	}
	return sm
}

// func (o Outcome) ID() uint32 {
// 	return toOutcomeID(o.SRID)
// }
// func (o Outcome) PlayerID() uint32 {
// 	return toPlayerID(o.SRID)
// }

// func (o *OddsChange) Markets() []Market {
// 	if o == nil || o.Odds == nil {
// 		return nil
// 	}
// 	return o.Odds.Markets
// }

// func (o *OddsChange) market(id uint32) *Market {
// 	for _, m := range o.Markets() {
// 		if *m.SRID == id {
// 			return &m
// 		}
// 	}
// 	return nil
// }