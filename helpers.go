package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

func getInvoice(label, desc string, msats int) (string, error) {
	res, err := ln.Call("listinvoices", label)
	if err != nil {
		return "", err
	}

	switch res.Get("invoices.0.status").String() {
	case "paid":
		return "", nil
	case "unpaid":
		bolt11 := res.Get("invoices.0.bolt11").String()
		return bolt11, nil
	case "expired":
		_, err := ln.Call("delinvoice", label, "expired")
		if err != nil {
			return "", err
		}
		res, err := ln.CallWithCustomTimeout("invoice", 7*time.Second, strconv.Itoa(msats), label, desc)
		if err != nil {
			return "", err
		}
		bolt11 := res.Get("bolt11").String()
		return bolt11, nil
	case "":
		res, err := ln.CallWithCustomTimeout("invoice", 7*time.Second, strconv.Itoa(msats), label, desc)
		if err != nil {
			return "", err
		}
		bolt11 := res.Get("bolt11").String()
		return bolt11, nil
	default:
		log.Warn().Str("label", label).Str("r", res.String()).
			Msg("what's up with this invoice?")
		return "", errors.New("bad wrong invoice")
	}
}

func checkPayment(label string, pricemsats int) (err error) {
	res, err := ln.Call("listinvoices", label)
	if err != nil {
		return
	}

	if res.Get("invoices.0.status").String() != "paid" {
		err = errors.New("invoice not paid")
		return
	}

	totalpaid := int(res.Get("invoices.0.msatoshi_received").Int())
	if totalpaid < pricemsats {
		err = fmt.Errorf("paid %d, needed %d", totalpaid, pricemsats)
		return
	}

	return nil
}

var reNumber = regexp.MustCompile("\\d+")

func stackTraceWithCode(stacktrace string, code string) string {
	var result []string

	stlines := strings.Split(stacktrace, "\n")
	lines := strings.Split(code, "\n")
	result = append(result, stlines[0])

	for i := 1; i < len(stlines); i++ {
		stline := stlines[i]
		result = append(result, stline)

		snum := reNumber.FindString(stline)
		if snum != "" {
			num, _ := strconv.Atoi(snum)
			for i, line := range lines {
				line = fmt.Sprintf("%3d %s", i+1, line)
				if i+1 > num-3 && i+1 < num+3 {
					result = append(result, line)
				}
			}
		}
	}

	return strings.Join(result, "\n")
}

func luaErrorType(apierr *lua.ApiError) string {
	switch apierr.Type {
	case lua.ApiErrorSyntax:
		return "ApiErrorSyntax"
	case lua.ApiErrorFile:
		return "ApiErrorFile"
	case lua.ApiErrorRun:
		return "ApiErrorRun"
	case lua.ApiErrorError:
		return "ApiErrorError"
	case lua.ApiErrorPanic:
		return "ApiErrorPanic"
	default:
		return "unknown"
	}
}