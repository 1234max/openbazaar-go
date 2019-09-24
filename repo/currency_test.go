package repo_test

import (
	"math/big"
	"strings"
	"testing"

	"github.com/OpenBazaar/openbazaar-go/repo"
	"github.com/OpenBazaar/openbazaar-go/test/factory"
)

func mustNewCurrencyValue(t *testing.T, amount, currencyCode string) *repo.CurrencyValue {
	var (
		def    = factory.NewCurrencyDefinition(currencyCode)
		c, err = repo.NewCurrencyValue(amount, def)
	)
	if err != nil {
		t.Fatalf("currency value (%s, %s): %s", amount, currencyCode, err.Error())
		return nil
	}
	return c
}

func TestCurrencyValueMarshalsToJSON(t *testing.T) {
	var (
		examples = []struct {
			value    string
			currency repo.CurrencyDefinition
		}{
			{ // valid currency value
				value:    "123456789012345678",
				currency: factory.NewCurrencyDefinition("ABC"),
			},
			{ // valid currency value large enough to overflow primative ints
				value:    "123456789012345678901234567890",
				currency: factory.NewCurrencyDefinition("BCD"),
			},
		}
	)

	for _, e := range examples {
		var (
			example, err = repo.NewCurrencyValue(e.value, e.currency)
			//actual       *repo.CurrencyValue
		)
		actual := &repo.CurrencyValue{}
		if err != nil {
			t.Errorf("unable to parse valid input '%s': %s", e.value, err.Error())
			continue
		}
		j, err := example.MarshalJSON()
		if err != nil {
			t.Errorf("marshaling %s: %s", example.String(), err)
			continue
		}

		if err := actual.UnmarshalJSON(j); err != nil {
			t.Errorf("unmarhsaling %s, %s", example.String(), err)
			continue
		}

		if !actual.Equal(example) {
			t.Errorf("expected %s and %s to be equal, but was not", example.String(), actual.String())
		}
	}
}

func TestCurrencyValuesAreValid(t *testing.T) {
	var examples = []struct {
		value       func() *repo.CurrencyValue
		expectedErr error
	}{
		{ // valid value
			value:       func() *repo.CurrencyValue { return mustNewCurrencyValue(t, "1", "BTC") },
			expectedErr: nil,
		},
		{ // invalid nil value
			value: func() *repo.CurrencyValue {
				var value = factory.MustNewCurrencyValue("123", "BTC")
				value.Amount = nil
				return value
			},
			expectedErr: repo.ErrCurrencyValueAmountInvalid,
		},
		{ // invalid nil currency
			value: func() *repo.CurrencyValue {
				var value = factory.MustNewCurrencyValue("123", "BTC")
				value.Currency = repo.NilCurrencyDefinition
				return value
			},
			expectedErr: repo.ErrCurrencyDefinitionUndefined,
		},
		{ // invalid currency definition makes value invalid (0 divisibility)
			value: func() *repo.CurrencyValue {
				var value = factory.MustNewCurrencyValue("123", "BTC")
				value.Currency.Divisibility = 0
				return value
			},
			expectedErr: repo.ErrCurrencyDivisibilityNonPositive,
		},
	}

	for _, e := range examples {
		var value = e.value()
		if err := value.Valid(); err != nil {
			if e.expectedErr != nil && err != e.expectedErr {
				t.Errorf("expected error (%s), but was (%s) for value (%s)", e.expectedErr.Error(), err.Error(), value.String())
			}
			if e.expectedErr == nil {
				t.Errorf("expected error to be nil, but was (%s) for value (%s)", err.Error(), value.String())
			}
			continue
		}
		if e.expectedErr != nil {
			t.Errorf("expected an error, but was nil for (%+v)", value)
			continue
		}
	}
}

func TestCurrencyValuesAreEqual(t *testing.T) {
	var examples = []struct {
		value    *repo.CurrencyValue
		other    *repo.CurrencyValue
		expected bool
	}{
		{ // value and currency divisibility different but
			// equal after normalizing
			value: &repo.CurrencyValue{
				Amount: big.NewInt(1234),
				Currency: repo.CurrencyDefinition{
					Code:         "BTC",
					Divisibility: 2,
					CurrencyType: "crypto",
				},
			},
			other: &repo.CurrencyValue{
				Amount: big.NewInt(123400),
				Currency: repo.CurrencyDefinition{
					Code:         "BTC",
					Divisibility: 4,
					CurrencyType: "crypto",
				},
			},
			expected: true,
		},
		{ // value and currency matching should be equal
			value: &repo.CurrencyValue{
				Amount:   big.NewInt(1),
				Currency: factory.NewCurrencyDefinition("BTC"),
			},
			other: &repo.CurrencyValue{
				Amount:   big.NewInt(1),
				Currency: factory.NewCurrencyDefinition("BTC"),
			},
			expected: true,
		},
		{ // nils should not be equal
			value:    nil,
			other:    nil,
			expected: false,
		},
		{ // nil should not match with a value
			value: nil,
			other: &repo.CurrencyValue{
				Amount:   big.NewInt(1),
				Currency: factory.NewCurrencyDefinition("BTC"),
			},
			expected: false,
		},
		{ // value should not match with nil
			value: &repo.CurrencyValue{
				Amount:   big.NewInt(1),
				Currency: factory.NewCurrencyDefinition("BTC"),
			},
			other:    nil,
			expected: false,
		},
		{ // value difference
			value: &repo.CurrencyValue{
				Amount:   big.NewInt(1),
				Currency: factory.NewCurrencyDefinition("BTC"),
			},
			other: &repo.CurrencyValue{
				Amount:   big.NewInt(2),
				Currency: factory.NewCurrencyDefinition("BTC"),
			},
			expected: false,
		},
		{ // currency code difference
			value: &repo.CurrencyValue{
				Amount:   big.NewInt(1),
				Currency: factory.NewCurrencyDefinition("BTC"),
			},
			other: &repo.CurrencyValue{
				Amount:   big.NewInt(1),
				Currency: factory.NewCurrencyDefinition("ETH"),
			},
			expected: false,
		},
		{ // currency code missing
			value: &repo.CurrencyValue{
				Amount:   big.NewInt(1),
				Currency: repo.NilCurrencyDefinition,
			},
			other: &repo.CurrencyValue{
				Amount:   big.NewInt(1),
				Currency: factory.NewCurrencyDefinition("ETH"),
			},
			expected: false,
		},
	}

	for _, c := range examples {
		if c.value.Equal(c.other) != c.expected {
			if c.expected {
				t.Errorf("expected %s to equal %s but did not", c.value.String(), c.other.String())
			} else {
				t.Errorf("expected %s to not equal %s but did", c.value.String(), c.other.String())
			}
		}

		// test that equal is communitive
		if c.other.Equal(c.value) != c.expected {
			if c.expected {
				t.Errorf("expected %s to equal %s but did not", c.other.String(), c.value.String())
			} else {
				t.Errorf("expected %s to not equal %s but did", c.other.String(), c.value.String())
			}
		}
	}
}

func TestCurrencyValuesConvertCorrectly(t *testing.T) {
	var (
		zeroRateErr             = "rate must be greater than zero"
		undefinedCurrencyErr    = "unknown currency"
		invalidErr              = "cannot convert invalid value"
		insufficentPrecisionErr = repo.ErrCurrencyValueInsufficientPrecision.Error()

		examples = []struct {
			value        *repo.CurrencyValue
			convertTo    repo.CurrencyDefinition
			exchangeRate float64
			expected     *repo.CurrencyValue
			expectedErr  *string
		}{
			{ // errors when definition is nil
				value:        factory.MustNewCurrencyValue("0", "BTC"),
				convertTo:    repo.NilCurrencyDefinition,
				exchangeRate: 0.99999,
				expected:     nil,
				expectedErr:  &undefinedCurrencyErr,
			},
			{ // errors zero rate
				value:        factory.MustNewCurrencyValue("123", "BTC"),
				convertTo:    factory.NewCurrencyDefinition("BCH"),
				exchangeRate: 0,
				expected:     nil,
				expectedErr:  &zeroRateErr,
			},
			{ // errors negative rate
				value:        factory.MustNewCurrencyValue("123", "BTC"),
				convertTo:    factory.NewCurrencyDefinition("BCH"),
				exchangeRate: -0.1,
				expected:     nil,
				expectedErr:  &zeroRateErr,
			},
			{ // rounds down
				value:        factory.MustNewCurrencyValue("1", "BTC"),
				convertTo:    factory.NewCurrencyDefinition("BCH"),
				exchangeRate: 0.9,
				expected:     factory.MustNewCurrencyValue("0", "BCH"),
				expectedErr:  nil,
			},
			{ // handles negative values
				value:        factory.MustNewCurrencyValue("-100", "BTC"),
				convertTo:    factory.NewCurrencyDefinition("BCH"),
				exchangeRate: 0.123,
				expected:     factory.MustNewCurrencyValue("-12", "BCH"),
				expectedErr:  nil,
			},
			{ // handles zero
				value:        factory.MustNewCurrencyValue("0", "BTC"),
				convertTo:    factory.NewCurrencyDefinition("BCH"),
				exchangeRate: 0.99999,
				expected:     factory.MustNewCurrencyValue("0", "BCH"),
				expectedErr:  nil,
			},
			{ // handles invalid value
				value: &repo.CurrencyValue{
					Amount:   big.NewInt(1000),
					Currency: repo.NilCurrencyDefinition,
				},
				convertTo:    factory.NewCurrencyDefinition("BTC"),
				exchangeRate: 0.5,
				expected:     nil,
				expectedErr:  &invalidErr,
			},
			{ // handles conversions between different divisibility
				value: &repo.CurrencyValue{
					Amount: big.NewInt(1000),
					Currency: repo.CurrencyDefinition{
						Name:         "United States Dollar",
						Code:         "USD",
						Divisibility: 2,
						CurrencyType: repo.Fiat,
					},
				},
				convertTo: repo.CurrencyDefinition{
					Name:         "SimpleCoin",
					Code:         "SPC",
					Divisibility: 6,
					CurrencyType: repo.Crypto,
				},
				exchangeRate: 0.5,
				expected: &repo.CurrencyValue{
					Amount: big.NewInt(5000000),
					Currency: repo.CurrencyDefinition{
						Name:         "SimpleCoin",
						Code:         "SPC",
						Divisibility: 6,
						CurrencyType: repo.Crypto,
					},
				},
				expectedErr: nil,
			},
			{ // handles conversions between different
				// divisibility
				value: &repo.CurrencyValue{ // 99.123456 SPC
					Amount: big.NewInt(99120000),
					Currency: repo.CurrencyDefinition{
						Name:         "SimpleCoin",
						Code:         "SPC",
						Divisibility: 6,
						CurrencyType: repo.Crypto,
					},
				},
				convertTo: repo.CurrencyDefinition{
					Name:         "United States Dollar",
					Code:         "USD",
					Divisibility: 2,
					CurrencyType: repo.Fiat,
				},
				exchangeRate: 2,
				expected: &repo.CurrencyValue{ // 99.123456 SPC * (2 USD/SPC)
					Amount: big.NewInt(19824),
					Currency: repo.CurrencyDefinition{
						Name:         "United States Dollar",
						Code:         "USD",
						Divisibility: 2,
						CurrencyType: repo.Fiat,
					},
				},
				expectedErr: nil,
			},
			{ // handles conversions which reduce significant figures
				// with an error
				value: &repo.CurrencyValue{
					Amount: big.NewInt(7654321),
					Currency: repo.CurrencyDefinition{
						Name:         "SimpleCoin",
						Code:         "SPC",
						Divisibility: 4,
						CurrencyType: repo.Crypto,
					},
				},
				convertTo: repo.CurrencyDefinition{
					Name:         "SimpleCoin",
					Code:         "SPC",
					Divisibility: 2,
					CurrencyType: repo.Crypto,
				},
				exchangeRate: 1,
				expected:     nil,
				expectedErr:  &insufficentPrecisionErr,
			},
		}
	)

	for _, e := range examples {
		actual, err := e.value.ConvertTo(e.convertTo, e.exchangeRate)
		if err != nil {
			if e.expectedErr != nil && !strings.Contains(err.Error(), *e.expectedErr) {
				t.Errorf("expected value (%s) to error with (%s) but returned: %s", e.value, *e.expectedErr, err.Error())
			}
			continue
		} else {
			if e.expectedErr != nil {
				t.Errorf("expected error (%s) but produced none", *e.expectedErr)
				t.Logf("\tfor value: (%s) convertTo: (%s) rate: (%f)", e.value, e.convertTo, e.exchangeRate)
			}
		}

		if !actual.Equal(e.expected) {
			t.Errorf("expected converted value to be %s, but was %s", e.expected, actual)
			t.Logf("\tfor value: (%s) convertTo: (%s) rate: (%f)", e.value, e.convertTo, e.exchangeRate)
			continue
		}
	}
}

func TestNewCurrencyValueWithLookup(t *testing.T) {
	_, err := repo.NewCurrencyValueWithLookup("0", "")
	if err == nil {
		t.Errorf("expected empty code to return an error, but did not")
	}

	_, err = repo.NewCurrencyValueWithLookup("0", "invalid")
	if err == nil {
		t.Errorf("expected invalid/undefined code to return an error, but did not")
	}

	subject, err := repo.NewCurrencyValueWithLookup("", "USD")
	if err != nil {
		t.Errorf("expected empty value to be accepted, but returned error: %s", err.Error())
	}
	if subject.String() != "0 United States Dollar (USDdiv2)" {
		t.Errorf("expected empty value to be set as (0 United State Dollar (USDdiv2)), but was (%s)", subject.String())
	}

	_, err = repo.NewCurrencyValueWithLookup("1234567890987654321", "ETH")
	if err != nil {
		t.Errorf("expected large value to be accepted, but returned error: %s", err.Error())
	}
}

func TestCurrencyValueAmount(t *testing.T) {
	subject := &repo.CurrencyValue{}
	actual := subject.AmountString()
	if actual != "0" {
		t.Errorf("expected zero value amount string to be (0), but was (%s)", actual)
	}

	subject = &repo.CurrencyValue{Amount: big.NewInt(100)}
	actual = subject.AmountString()
	if actual != "100" {
		t.Errorf("expected set value to be (%s), but was (%s)", "100", actual)
	}
}

func TestCurrencyValueAdjustDivisibility(t *testing.T) {
	sameDiv := uint(8)
	subject := factory.MustNewCurrencyValue("123000000", "BTC")
	subject.Currency.Divisibility = sameDiv

	if newValue, err := subject.AdjustDivisibility(sameDiv); err != nil {
		t.Fatalf("expected same divisibility to not return an error, but did: %s", err.Error())
	} else {
		if !newValue.Currency.Equal(subject.Currency) {
			t.Errorf("expected same divisibility to produce equal currencies, but did not")
		}
	}

	if newValue, err := subject.AdjustDivisibility(2); err != nil {
		t.Fatalf("expected new divisibility to not return an error, but did: %s", err.Error())
	} else {
		if newValue.Currency.Equal(subject.Currency) {
			t.Errorf("expected new divisibility to produce different currency, but did not")
		}
	}
}
