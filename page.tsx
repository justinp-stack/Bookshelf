"use client";

import { useState, useEffect, Suspense } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";

interface BookType {
  id: number;
  title: string;
}

// Custom type for our new sleek notifications
interface NotificationType {
  text: string;
  type: "success" | "error";
}

function CheckoutLedger() {
  const searchParams = useSearchParams();
  const incomingTitle = searchParams.get("title") || "";

  const [student, setStudent] = useState("");
  const [books, setBooks] = useState<string[]>(incomingTitle ? [incomingTitle] : [""]);
  const [availableBooks, setAvailableBooks] = useState<BookType[]>([]);
  
  const [today, setToday] = useState("");
  const [dueDate, setDueDate] = useState("");
  const [showStamp, setShowStamp] = useState(false);

  // NEW: State for our custom styled notifications
  const [notification, setNotification] = useState<NotificationType | null>(null);

  // NEW: Helper function to show notifications that auto-hide after 4 seconds
  const showNotification = (text: string, type: "success" | "error") => {
    setNotification({ text, type });
    setTimeout(() => {
      setNotification(null);
    }, 4000);
  };

  useEffect(() => {
    const currentDate = new Date();
    setToday(currentDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }));
    
    const returnDate = new Date();
    returnDate.setDate(returnDate.getDate() + 21);
    setDueDate(returnDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }).toUpperCase());

    const fetchShelf = async () => {
      const storedKey = localStorage.getItem("my_app_jwt_token");
      if (!storedKey) return; 

      try {
        const res = await fetch("http://localhost:8080/books", {
          headers: { "Authorization": `Bearer ${storedKey}` },
        });
        
        if (res.ok) {
          const data = await res.json();
          const fetchedBooks = Array.isArray(data) ? data : (data.books || []);
          
          fetchedBooks.sort((a: BookType, b: BookType) => 
            (a.title || "").localeCompare(b.title || "")
          );
          
          setAvailableBooks(fetchedBooks);
        }
      } catch (error) {
        console.error("Failed to load bookshelf", error);
      }
    };

    fetchShelf();
  }, []);

  function handleBookChange(index: number, value: string) {
    const newBooks = [...books];
    newBooks[index] = value;
    setBooks(newBooks);
  }

  function addBookRow() {
    setBooks([...books, ""]);
  }

  function removeBookRow(index: number) {
    if (books.length > 1) {
      setBooks(books.filter((_, i) => i !== index));
    }
  }

  async function handleCheckout(e: React.FormEvent) {
    e.preventDefault();
    if (!student.trim() || books.every(book => !book.trim())) return;

    const storedKey = localStorage.getItem("my_app_jwt_token");
    if (!storedKey) {
      showNotification("Missing Token. Please ensure you are logged in.", "error");
      return;
    }

    let allSuccessful = true;

    for (const bookTitle of books) {
      if (!bookTitle.trim()) continue;

      const selectedBook = availableBooks.find(b => b.title === bookTitle);
      if (!selectedBook) continue;

      try {
        const res = await fetch(`http://localhost:8080/books/${selectedBook.id}/borrow`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "Authorization": `Bearer ${storedKey}`,
          },
        });

        if (!res.ok) {
          const errorData = await res.json();
          showNotification(`Could not check out "${bookTitle}": ${errorData.error}`, "error");
          allSuccessful = false;
        }
      } catch (error) {
        console.error("Failed to process checkout:", error);
        showNotification(`Network error while trying to check out "${bookTitle}".`, "error");
        allSuccessful = false;
      }
    }

    if (allSuccessful) {
      setShowStamp(true);
    }
  }

  async function handleReturn(e: React.MouseEvent) {
    e.preventDefault();
    
    if (books.every(book => !book.trim())) {
      showNotification("Please select at least one book to return.", "error");
      return;
    }

    const storedKey = localStorage.getItem("my_app_jwt_token");
    if (!storedKey) {
      showNotification("Missing Token. Please ensure you are logged in.", "error");
      return;
    }

    let allSuccessful = true;
    let returnCount = 0;

    for (const bookTitle of books) {
      if (!bookTitle.trim()) continue;

      const selectedBook = availableBooks.find(b => b.title === bookTitle);
      if (!selectedBook) continue;

      try {
        const res = await fetch(`http://localhost:8080/books/${selectedBook.id}/return`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "Authorization": `Bearer ${storedKey}`,
          },
        });

        if (!res.ok) {
          const errorData = await res.json();
          showNotification(`Could not return "${bookTitle}": ${errorData.error}`, "error");
          allSuccessful = false;
        } else {
          returnCount++;
        }
      } catch (error) {
        console.error("Failed to process return:", error);
        showNotification(`Network error while trying to return "${bookTitle}".`, "error");
        allSuccessful = false;
      }
    }

    if (allSuccessful && returnCount > 0) {
      showNotification(`Success! Returned ${returnCount} material(s) to the shelf.`, "success");
      setBooks([""]);
      setStudent("");
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center bg-gray-200 p-8 overflow-hidden">
      
      {/* NEW: Custom Styled Notification Toast */}
      {notification && (
        <div className={`fixed top-8 left-1/2 -translate-x-1/2 z-[100] px-6 py-4 border-[3px] border-black font-bold uppercase tracking-widest shadow-[6px_6px_0px_0px_rgba(0,0,0,1)] transition-all animate-[slideDown_0.3s_ease-out] flex items-center justify-between gap-6 min-w-[320px] max-w-xl ${
          notification.type === 'error' ? 'bg-red-400 text-black' : 'bg-green-400 text-black'
        }`}>
          <span>{notification.text}</span>
          <button 
            onClick={() => setNotification(null)}
            className="text-2xl hover:scale-110 transition-transform cursor-pointer"
          >
            X
          </button>
        </div>
      )}

      {/* The Giant Middle Page Stamp */}
      {showStamp && (
        <div className="absolute inset-0 z-50 flex items-center justify-center pointer-events-auto bg-black/10 backdrop-blur-sm transition-all">
          <div className="animate-[stamp_0.3s_ease-in_forwards] relative flex flex-col items-center justify-center border-8 border-red-600 px-12 py-8 rotate-[-12deg] bg-white/90 shadow-2xl">
            
            <button 
              onClick={() => setShowStamp(false)}
              className="absolute top-2 right-4 text-red-600 font-mono text-3xl font-black hover:scale-125 transition-transform cursor-pointer"
              aria-label="Close stamp"
              type="button"
            >
              X
            </button>

            <span className="text-red-600 font-mono text-3xl font-bold tracking-widest uppercase mb-2 mt-4">DATE DUE</span>
            <span className="text-red-600 font-mono text-6xl font-black tracking-tighter uppercase">{dueDate}</span>
            <span className="text-red-600 font-mono text-xl font-bold tracking-widest uppercase mt-4 border-t-4 border-red-600 pt-2">RETURN TO LIBRARY</span>
            <span className="text-red-600 font-mono text-md font-bold tracking-widest uppercase mt-2">
              ({books.filter(b => b.trim()).length} ITEM{books.filter(b => b.trim()).length !== 1 ? 'S' : ''} CHECKED OUT)
            </span>
          </div>
        </div>
      )}

      {/* The Worksheet Paper */}
      <div className={`w-full max-w-4xl bg-white border-[3px] border-black rounded-lg shadow-xl p-8 pb-12 transition-all ${showStamp ? 'opacity-40 blur-sm pointer-events-none' : ''}`}>
        
        {/* Header Section */}
        <div className="flex items-center justify-between mb-12 px-4 border-b-4 border-black pb-8">
          <div className="text-left">
            <h1 className="text-4xl sm:text-5xl font-black tracking-wider text-black">
              CHECKOUT DESK
            </h1>
            <p className="text-sm text-gray-600 mt-2 uppercase tracking-widest font-mono">
              Policy: All materials are due exactly 3 weeks from Date Out
            </p>
          </div>
          <Link href="/" className="px-4 py-2 bg-gray-200 hover:bg-gray-300 text-black font-bold uppercase tracking-widest text-sm border-2 border-black rounded-sm transition-colors shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] active:shadow-none active:translate-x-1 active:translate-y-1">
            ← Back to Shelf
          </Link>
        </div>

        {/* The Checkout Form */}
        <form onSubmit={handleCheckout} className="space-y-8 max-w-2xl mx-auto">
          
          <div className="space-y-2">
            <label className="text-xl font-bold uppercase tracking-widest text-black">Patron Name</label>
            <input
              type="text"
              placeholder="e.g. John Doe"
              value={student}
              onChange={(e) => setStudent(e.target.value)}
              className="w-full bg-gray-50 border-[3px] border-black outline-none px-4 py-4 text-xl placeholder:text-gray-400 font-medium focus:bg-blue-50 transition-colors"
              required
            />
          </div>

          <div className="space-y-4">
            <div className="flex justify-between items-end">
              <label className="text-xl font-bold uppercase tracking-widest text-black">Selected Material(s)</label>
              <button 
                type="button" 
                onClick={addBookRow}
                className="text-sm font-bold uppercase tracking-widest text-black border-2 border-black px-3 py-1 hover:bg-gray-200 transition-colors shadow-[2px_2px_0px_0px_rgba(0,0,0,1)] active:shadow-none active:translate-x-0.5 active:translate-y-0.5"
              >
                + Add Another
              </button>
            </div>
            
            <div className="space-y-3">
              {books.map((book, index) => (
                <div key={index} className="flex items-center gap-2">
                  <span className="font-mono font-bold text-lg w-8 text-center">{index + 1}.</span>
                  
                  <select
                    value={book}
                    onChange={(e) => handleBookChange(index, e.target.value)}
                    className="w-full bg-gray-50 border-[3px] border-black outline-none px-4 py-3 text-xl font-medium focus:bg-blue-50 transition-colors cursor-pointer"
                    required
                  >
                    <option value="" disabled>-- SELECT A BOOK FROM SHELF --</option>
                    {availableBooks.map((b) => (
                      <option key={b.id} value={b.title}>
                        {b.title}
                      </option>
                    ))}
                    
                    {book && !availableBooks.find(b => b.title === book) && (
                      <option value={book}>{book}</option>
                    )}
                  </select>

                  {books.length > 1 && (
                    <button
                      type="button"
                      onClick={() => removeBookRow(index)}
                      className="shrink-0 flex items-center justify-center w-12 h-[52px] bg-white border-[3px] border-black text-black font-black text-xl hover:bg-red-200 hover:text-red-700 hover:border-red-700 transition-colors"
                      title="Remove Book"
                    >
                      X
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-8 pt-4">
            <div className="flex flex-col space-y-2">
              <span className="text-sm font-bold uppercase tracking-widest text-gray-500">Date Out</span>
              <span className="text-2xl font-mono font-bold text-black border-b-2 border-black pb-2">{today}</span>
            </div>
            <div className="flex flex-col space-y-2">
              <span className="text-sm font-bold uppercase tracking-widest text-gray-500">Calculated Return</span>
              <span className="text-2xl font-mono font-bold text-gray-400 border-b-2 border-gray-300 pb-2 italic">Upon stamping...</span>
            </div>
          </div>

          <div className="flex flex-col space-y-4 mt-8">
            <button
              type="submit"
              className="w-full bg-black text-white text-2xl uppercase tracking-widest font-black py-6 hover:bg-gray-800 transition-colors shadow-[8px_8px_0px_0px_rgba(0,0,0,0.2)] active:shadow-none active:translate-x-2 active:translate-y-2 cursor-pointer"
            >
              CHECKOUT
            </button>
            
            <button
              type="button"
              onClick={handleReturn}
              className="w-full bg-transparent border-[3px] border-black text-black text-lg uppercase tracking-widest font-bold py-3 hover:bg-gray-100 transition-colors shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] active:shadow-none active:translate-x-1 active:translate-y-1 cursor-pointer"
            >
              RETURN
            </button>
          </div>
        </form>
      </div>

      <style dangerouslySetInnerHTML={{__html: `
        @keyframes stamp {
          0% { transform: scale(3) rotate(0deg); opacity: 0; }
          50% { transform: scale(0.8) rotate(-15deg); opacity: 1; }
          100% { transform: scale(1) rotate(-12deg); opacity: 1; }
        }
        @keyframes slideDown {
          from { transform: translate(-50%, -100%); opacity: 0; }
          to { transform: translate(-50%, 0); opacity: 1; }
        }
      `}} />
    </div>
  );
}

export default function CheckoutPage() {
  return (
    <Suspense fallback={<div className="min-h-screen flex items-center justify-center font-mono text-xl">Loading records...</div>}>
      <CheckoutLedger />
    </Suspense>
  );
}