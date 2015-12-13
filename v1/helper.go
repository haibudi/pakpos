package pakpos

import (
	"fmt"
	. "github.com/eaciit/toolkit"
)

func CallResult(url, calltype string, data []byte) (*Result, error) {
	r, e := HttpCall(url, calltype, data, M{}.Set("expectedstatus", 200))
	if e != nil {
		return nil, fmt.Errorf(url + " Call eror: " + e.Error())
	}

	result := NewResult()
	edecode := Unjson(HttpContent(r), &result)
	if edecode != nil {
		return nil, fmt.Errorf(url + " Decode error: " + edecode.Error())
	}
	if result.Status == Status_NOK {
		return result, fmt.Errorf(result.Message)
	}
	return result, nil
}
