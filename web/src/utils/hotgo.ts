import { Ref, UnwrapRef } from '@vue/reactivity';
import onerrorImg from '@/assets/images/onerror.png';

export interface Option {
  label: string;
  value: string | number;
  key: string | number;
  // type: string;
  listClass: 'default' | 'error' | 'primary' | 'info' | 'success' | 'warning';
}

export interface Options {
  [name: string]: Option[];
}

// 获取选项名称
export function getOptionLabel(options: Option[], value) {
  if (options === undefined || options?.length === 0) {
    return `unknown`;
  }
  for (const item of options) {
    if (item.value == value) {
      return item.label;
    }
  }

  return `unknown`;
}

// 获取选项标签
export function getOptionTag(options: Option[], value) {
  if (options === undefined || options?.length === 0) {
    return 'default';
  }
  for (const item of options) {
    if (item.value == value) {
      return item.listClass;
    }
  }

  return 'default';
}

// 自适应模板宽度
export function adaModalWidth(dialogWidth: Ref<UnwrapRef<string>>) {
  const val = document.body.clientWidth;
  const def = 840; // 默认宽度
  if (val <= def) {
    dialogWidth.value = '100%';
  } else {
    dialogWidth.value = def + 'px';
  }
  return dialogWidth.value;
}

// 图片加载失败显示自定义默认图片(缺省图)
export function errorImg(e: any): void {
  e.target.src = onerrorImg;
  e.target.onerror = null;
}
