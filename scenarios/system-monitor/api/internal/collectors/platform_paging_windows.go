//go:build windows

package collectors

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// PDH is called through pdh.dll procedures; no cgo. The query stays open for
// the life of the process so each collection costs one PdhCollectQueryData.
var (
	modPDH                          = windows.NewLazySystemDLL("pdh.dll")
	procPdhOpenQueryW               = modPDH.NewProc("PdhOpenQueryW")
	procPdhAddEnglishCounterW       = modPDH.NewProc("PdhAddEnglishCounterW")
	procPdhCollectQueryData         = modPDH.NewProc("PdhCollectQueryData")
	procPdhGetRawCounterValue       = modPDH.NewProc("PdhGetRawCounterValue")
	procPdhGetFormattedCounterValue = modPDH.NewProc("PdhGetFormattedCounterValue")

	modKernel32Pressure                  = windows.NewLazySystemDLL("kernel32.dll")
	procCreateMemoryResourceNotification = modKernel32Pressure.NewProc("CreateMemoryResourceNotification")
	procQueryMemoryResourceNotification  = modKernel32Pressure.NewProc("QueryMemoryResourceNotification")
)

const (
	pdhFmtDouble                  = 0x00000200
	pdhCStatusValidData           = 0x00000000
	pdhCStatusNewData             = 0x00000001
	lowMemoryResourceNotification = 0
)

// pdhRawCounter mirrors PDH_RAW_COUNTER (40 bytes on amd64/arm64).
type pdhRawCounter struct {
	CStatus     uint32
	TimeStamp   windows.Filetime
	FirstValue  int64
	SecondValue int64
	MultiCount  uint32
}

// pdhFmtCounterValueDouble mirrors PDH_FMT_COUNTERVALUE with the double arm
// of the union.
type pdhFmtCounterValueDouble struct {
	CStatus     uint32
	_           uint32
	DoubleValue float64
}

type windowsPDHQuery struct {
	mu         sync.Mutex
	opened     bool
	openErr    error
	query      uintptr
	pagesInput uintptr
	pageReads  uintptr
	pagingFile uintptr
	addErrs    map[string]error
	lowMemory  uintptr
	lowErr     error
}

var windowsPressureQuery windowsPDHQuery

func pdhStatus(ret uintptr) error {
	if ret == 0 {
		return nil
	}
	return fmt.Errorf("PDH status 0x%08x", uint32(ret))
}

func (q *windowsPDHQuery) open() {
	if q.opened {
		return
	}
	q.opened = true
	q.addErrs = make(map[string]error)
	if err := procPdhOpenQueryW.Find(); err != nil {
		q.openErr = err
		return
	}
	if err := pdhStatus(callProc(procPdhOpenQueryW, 0, 0, uintptr(unsafe.Pointer(&q.query)))); err != nil {
		q.openErr = fmt.Errorf("PdhOpenQueryW: %w", err)
		return
	}
	add := func(path string) uintptr {
		var counter uintptr
		ptr, err := windows.UTF16PtrFromString(path)
		if err == nil {
			err = pdhStatus(callProc(procPdhAddEnglishCounterW, q.query, uintptr(unsafe.Pointer(ptr)), 0, uintptr(unsafe.Pointer(&counter))))
		}
		if err != nil {
			q.addErrs[path] = fmt.Errorf("PdhAddEnglishCounterW(%s): %w", path, err)
			return 0
		}
		return counter
	}
	q.pagesInput = add(`\Memory\Pages Input/sec`)
	q.pageReads = add(`\Memory\Page Reads/sec`)
	q.pagingFile = add(`\Paging File(_Total)\% Usage`)

	handle, _, callErr := procCreateMemoryResourceNotification.Call(lowMemoryResourceNotification)
	if handle == 0 {
		q.lowErr = fmt.Errorf("CreateMemoryResourceNotification: %v", callErr)
	} else {
		q.lowMemory = handle
	}
}

func callProc(proc *windows.LazyProc, args ...uintptr) uintptr {
	ret, _, _ := proc.Call(args...)
	return ret
}

func (q *windowsPDHQuery) rawCounter(counter uintptr, path string) (uint64, error) {
	if counter == 0 {
		if err := q.addErrs[path]; err != nil {
			return 0, err
		}
		return 0, errors.New("counter unavailable")
	}
	var value pdhRawCounter
	var counterType uint32
	if err := pdhStatus(callProc(procPdhGetRawCounterValue, counter, uintptr(unsafe.Pointer(&counterType)), uintptr(unsafe.Pointer(&value)))); err != nil {
		return 0, fmt.Errorf("PdhGetRawCounterValue: %w", err)
	}
	if value.CStatus != pdhCStatusValidData && value.CStatus != pdhCStatusNewData {
		return 0, fmt.Errorf("counter status 0x%08x", value.CStatus)
	}
	if value.FirstValue < 0 {
		return 0, fmt.Errorf("negative raw counter %d", value.FirstValue)
	}
	return uint64(value.FirstValue), nil
}

func (q *windowsPDHQuery) formattedDouble(counter uintptr, path string) (float64, error) {
	if counter == 0 {
		if err := q.addErrs[path]; err != nil {
			return 0, err
		}
		return 0, errors.New("counter unavailable")
	}
	var value pdhFmtCounterValueDouble
	var counterType uint32
	if err := pdhStatus(callProc(procPdhGetFormattedCounterValue, counter, pdhFmtDouble, uintptr(unsafe.Pointer(&counterType)), uintptr(unsafe.Pointer(&value)))); err != nil {
		return 0, fmt.Errorf("PdhGetFormattedCounterValue: %w", err)
	}
	if value.CStatus != pdhCStatusValidData && value.CStatus != pdhCStatusNewData {
		return 0, fmt.Errorf("counter status 0x%08x", value.CStatus)
	}
	return value.DoubleValue, nil
}

func (q *windowsPDHQuery) sample() windowsPagingSample {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.open()
	var sample windowsPagingSample
	if q.openErr != nil {
		sample.PagesInputErr, sample.PageReadsErr, sample.PagingFileErr = q.openErr, q.openErr, q.openErr
	} else if err := pdhStatus(callProc(procPdhCollectQueryData, q.query)); err != nil {
		collectErr := fmt.Errorf("PdhCollectQueryData: %w", err)
		sample.PagesInputErr, sample.PageReadsErr, sample.PagingFileErr = collectErr, collectErr, collectErr
	} else {
		sample.PagesInputTotal, sample.PagesInputErr = q.rawCounter(q.pagesInput, `\Memory\Pages Input/sec`)
		sample.PageReadsTotal, sample.PageReadsErr = q.rawCounter(q.pageReads, `\Memory\Page Reads/sec`)
		sample.PagingFileUsagePercent, sample.PagingFileErr = q.formattedDouble(q.pagingFile, `\Paging File(_Total)\% Usage`)
	}

	status := windowsMemoryStatusEx{Length: uint32(unsafe.Sizeof(windowsMemoryStatusEx{}))}
	if ret, _, callErr := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status))); ret == 0 {
		sample.CommitErr = fmt.Errorf("GlobalMemoryStatusEx: %v", callErr)
	} else {
		sample.CommitLimit = status.TotalPageFile
		if status.TotalPageFile >= status.AvailPageFile {
			sample.CommitCharge = status.TotalPageFile - status.AvailPageFile
		}
	}

	if q.lowErr != nil {
		sample.LowMemoryErr = q.lowErr
	} else {
		var state int32
		if ret, _, callErr := procQueryMemoryResourceNotification.Call(q.lowMemory, uintptr(unsafe.Pointer(&state))); ret == 0 {
			sample.LowMemoryErr = fmt.Errorf("QueryMemoryResourceNotification: %v", callErr)
		} else {
			sample.LowMemory = state != 0
		}
	}
	return sample
}

func readPlatformPaging(_ map[string]uint64) pagingReading {
	reading := windowsPagingReading(windowsPressureQuery.sample())
	reading.pageSize = uint64(windows.Getpagesize())
	return reading
}
