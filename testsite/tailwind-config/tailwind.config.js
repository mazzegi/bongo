const path = require('path');
const siteRoot = path.join(__dirname, '..');
const relevantFilesGlob = '**/*.html';
const colors = require('tailwindcss/colors')

function withOpacityValue(variable) {
  return ({ opacityValue }) => {
    if (opacityValue === undefined) {
      return `rgb(var(${variable}))`
    }
    return `rgb(var(${variable}) / ${opacityValue})`
  }
}

module.exports = {
  mode: 'jit',
  content: [path.join(siteRoot, relevantFilesGlob)],
  theme: {
    colors: {
      current: colors.current,
      transparent: colors.transparent,
      black: colors.black,
      white: colors.white,
      gray: colors.gray,
      primary: withOpacityValue('--color-primary'),
      secondary: withOpacityValue('--color-secondary'),
    }
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
