package winapi

import (
	"fmt"
	"syscall"
	"unsafe"
)

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
