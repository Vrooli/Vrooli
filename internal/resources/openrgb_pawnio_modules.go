package resources

// Exact data modules in the complete official source. The HID serving recipe
// verifies this source first, then omits precisely these executable-adjacent files.
type openRGBPawnIOSourceModule struct {
	SHA256 string
	Bytes  int64
}

func openRGBPawnIOSourceModules() map[string]openRGBPawnIOSourceModule {
	return map[string]openRGBPawnIOSourceModule{
		"LedsValve.bin":            {"1c1809281ae6f3e3c5b362f1536c0e6706013e6e33bfba66cad74636e61436d6", 43796},
		"LpcIO.bin":                {"b3896a1cab0d808fca31fe2ebcae045d59dac690da87b17c858bb8da357eb45e", 18076},
		"SmbusI801.bin":            {"a0f7d066e7efda28c0e754c1f52dbd8dc280d388ba8575b514a80cb67490530c", 55540},
		"SmbusIntelSkylakeIMC.bin": {"2e2d6c1e541fdfe598aa8df4d334e11c491c8a84f23e8256dd18119dd420dbd2", 19828},
		"SmbusNCT6793.bin":         {"db068c6a87c0066ebfb7f691d88422fee8af350898e7a7f0db100fd6632bab17", 21244},
		"SmbusPIIX4.bin":           {"91f9b4b1c39e3d399ce48477a89d8f6bd3e58a2241064a4daec3a1513dff56e5", 42404},
	}
}
