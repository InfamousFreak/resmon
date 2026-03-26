import type { Metadata } from "next";
import { Space_Mono } from "next/font/google";
import "./globals.css";

const spaceMono = Space_Mono({
  weight: ['400', '700'],
  subsets: ["latin"]
});

export const metadata: Metadata = {
  title: "AI Research Monitor",
  description: "Real-time efficiency tracking for ML papers",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className={`${spaceMono.className} bg-black text-green-500`}>
        {children}
      </body>
    </html>
  );
}