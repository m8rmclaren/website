import type { Config } from "tailwindcss";

const config: Config = {
    corePlugins: {
        // preflight: false,
    },
    content: ["./components/**/*.templ"],
    theme: {
        extend: {
            backgroundImage: {
                "gradient-radial": "radial-gradient(var(--tw-gradient-stops))",
                "gradient-conic":
                    "conic-gradient(from 180deg at 50% 50%, var(--tw-gradient-stops))",
            },
            textColor: {
                primary: '#000000', // Black
                secondary: '#4A4A4A', // Dark Gray
                danger: '#e3342f',
            },
            backgroundColor: {
                'primary': '#3490dc',
                'secondary': '#ffed4a',
                'accent-1': '#5F60F6',
                'accent-2': '#F09878',
            },
            borderColor: {
                'accent-1': '#5F60F6',
                'accent-2': '#F09878',
            },
            keyframes: {
                shimmer: {
                    '0%': { transform: 'translateX(-100%)' },
                    '100%': { transform: 'translateX(100%)' },
                },
            },
            animation: {
                shimmer: 'shimmer 2s infinite',
            },
        },
    },
    plugins: [],
};
export default config;

