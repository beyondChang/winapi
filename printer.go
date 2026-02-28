package winapi

import (
	"fmt"
	"syscall"
	"unsafe"
)

func ListPrinters() ([]string, string) {
	flags := PRINTER_ENUM_LOCAL | PRINTER_ENUM_CONNECTIONS
	var needed uint32
	var returned uint32
	_ = EnumPrinters(uint32(flags), nil, 4, nil, 0, &needed, &returned)
	printers := []string{}
	if needed > 0 {
		buf := make([]byte, needed)
		if EnumPrinters(uint32(flags), nil, 4, &buf[0], needed, &needed, &returned) {
			entrySize := int(unsafe.Sizeof(PRINTER_INFO_4{}))
			for i := uint32(0); i < returned; i++ {
				offset := int(i) * entrySize
				if offset+entrySize <= len(buf) {
					pi4 := (*PRINTER_INFO_4)(unsafe.Pointer(&buf[offset]))
					name := UTF16PtrToString(pi4.PPrinterName)
					if name != "" {
						printers = append(printers, name)
					}
				}
			}
		}
	}
	var defNeeded uint32
	_ = GetDefaultPrinter(nil, &defNeeded)
	def := ""
	if defNeeded > 0 {
		defBuf := make([]uint16, defNeeded)
		if GetDefaultPrinter(&defBuf[0], &defNeeded) {
			def = syscall.UTF16ToString(defBuf)
		}
	}
	return printers, def
}

// PrintToPrinter 封装打印流程
// printerName: 打印机名称（可选，空字符串则用默认打印机）
// printFunc: 打印内容回调，参数为 hdc
func Print(printerName string, printFunc func(hdc HDC)) error {
	// 获取打印机名称
	if printerName == "" {
		var needed uint32 = 0
		GetDefaultPrinter(nil, &needed)
		if needed == 0 {
			return fmt.Errorf("未找到默认打印机")
		}
		buf := make([]uint16, needed)
		if !GetDefaultPrinter(&buf[0], &needed) {
			return fmt.Errorf("获取默认打印机失败")
		}
		printerName = syscall.UTF16ToString(buf)
	}

	// 转换打印机名称为 *uint16
	printerNamePtr, err := syscall.UTF16PtrFromString(printerName)
	if err != nil {
		return fmt.Errorf("打印机名称转换失败: %v", err)
	}
	// 创建打印机设备环境（HDC）
	driver, _ := syscall.UTF16PtrFromString("WINSPOOL")
	var output *uint16 = nil
	var devmode *DEVMODE = nil
	hdc := CreateDC(driver, printerNamePtr, output, devmode)
	if hdc == 0 {
		return fmt.Errorf("创建打印机设备环境失败,请检查打印机是否连接")
	}
	defer DeleteDC(hdc)

	// 设置文档信息
	docName, _ := syscall.UTF16PtrFromString("Go打印文档")
	docInfo := DOCINFO{
		CbSize:      int32(unsafe.Sizeof(DOCINFO{})),
		LpszDocName: docName,
	}

	// 开始打印作业
	if StartDoc(hdc, &docInfo) <= 0 {
		return fmt.Errorf("开始打印作业失败")
	}
	// 开始页面
	if StartPage(hdc) <= 0 {
		EndDoc(hdc)
		return fmt.Errorf("开始打印页面失败")
	}

	// 打印内容由回调函数实现
	printFunc(hdc)

	// 结束页面
	EndPage(hdc)
	// 结束文档
	EndDoc(hdc)
	return nil
}

func PrintWithPage(printerName string, widthMM int, heightMM int, landscape bool, printFunc func(hdc HDC)) error {
	if printerName == "" {
		var needed uint32 = 0
		GetDefaultPrinter(nil, &needed)
		if needed == 0 {
			return fmt.Errorf("未找到默认打印机")
		}
		buf := make([]uint16, needed)
		if !GetDefaultPrinter(&buf[0], &needed) {
			return fmt.Errorf("获取默认打印机失败")
		}
		printerName = syscall.UTF16ToString(buf)
	}
	printerNamePtr, err := syscall.UTF16PtrFromString(printerName)
	if err != nil {
		return fmt.Errorf("打印机名称转换失败: %v", err)
	}
	var dm DEVMODE
	dm.DmSize = uint16(unsafe.Sizeof(DEVMODE{}))
	dm.DmFields = DM_PAPERWIDTH | DM_PAPERLENGTH | DM_ORIENTATION
	dm.DmPaperWidth = int16(widthMM * 10)
	dm.DmPaperLength = int16(heightMM * 10)
	dm.DmOrientation = DMORIENT_PORTRAIT
	if landscape {
		dm.DmOrientation = DMORIENT_LANDSCAPE
	}
	driver, _ := syscall.UTF16PtrFromString("WINSPOOL")
	var output *uint16 = nil
	hdc := CreateDC(driver, printerNamePtr, output, &dm)
	if hdc == 0 {
		return fmt.Errorf("创建打印机设备环境失败,请检查打印机是否连接")
	}
	defer DeleteDC(hdc)
	docName, _ := syscall.UTF16PtrFromString("Go打印文档")
	docInfo := DOCINFO{
		CbSize:      int32(unsafe.Sizeof(DOCINFO{})),
		LpszDocName: docName,
	}
	if StartDoc(hdc, &docInfo) <= 0 {
		return fmt.Errorf("开始打印作业失败")
	}
	if StartPage(hdc) <= 0 {
		EndDoc(hdc)
		return fmt.Errorf("开始打印页面失败")
	}
	printFunc(hdc)
	EndPage(hdc)
	EndDoc(hdc)
	return nil
}
