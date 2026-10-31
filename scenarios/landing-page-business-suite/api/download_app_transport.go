package main

import downloadhttp "landing-page-business-suite-api/handlers/delivery"

func deliveryAppDependencies(plans interface{ BundleKey() string }) downloadhttp.AppDependencies {
	return downloadhttp.AppDependencies{
		BundleKey: plans.BundleKey, PathParam: getPathParam, DecodeJSON: decodeJSONBody,
		WriteError: writeJSONError, WriteData: writeJSONSuccessData, WriteSuccess: writeJSONSuccessSimple,
	}
}
