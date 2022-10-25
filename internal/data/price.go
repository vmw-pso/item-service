package data

import (
	"errors"
	"strconv"
	"unicode/utf8"
)

var ErrInvalidPriceFormat = errors.New("invalid price format")

type Price float64

func (p *Price) UnmarshalJSON(jsonValue []byte) error {
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidPriceFormat
	}

	price := trimFirstRune(unquotedJSONValue)

	f, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return ErrInvalidPriceFormat
	}

	*p = Price(f)

	return nil
}

func trimFirstRune(s string) string {
	_, i := utf8.DecodeRuneInString(s)
	return s[i:]
}
