export const auth_walletComplete = {
  "fieldName": "walletComplete",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "walletComplete",
        "loc": {
          "start": 8419,
          "end": 8433
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8434,
              "end": 8439
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 8442,
                "end": 8447
              }
            },
            "loc": {
              "start": 8441,
              "end": 8447
            }
          },
          "loc": {
            "start": 8434,
            "end": 8447
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "firstLogIn",
              "loc": {
                "start": 8455,
                "end": 8465
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8455,
              "end": 8465
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "session",
              "loc": {
                "start": 8470,
                "end": 8477
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Session_full",
                    "loc": {
                      "start": 8491,
                      "end": 8503
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8488,
                    "end": 8503
                  }
                }
              ],
              "loc": {
                "start": 8478,
                "end": 8509
              }
            },
            "loc": {
              "start": 8470,
              "end": 8509
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wallet",
              "loc": {
                "start": 8514,
                "end": 8520
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Wallet_common",
                    "loc": {
                      "start": 8534,
                      "end": 8547
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8531,
                    "end": 8547
                  }
                }
              ],
              "loc": {
                "start": 8521,
                "end": 8553
              }
            },
            "loc": {
              "start": 8514,
              "end": 8553
            }
          }
        ],
        "loc": {
          "start": 8449,
          "end": 8557
        }
      },
      "loc": {
        "start": 8419,
        "end": 8557
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 40,
          "end": 42
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 40,
        "end": 42
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 43,
          "end": 53
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 43,
        "end": 53
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 54,
          "end": 64
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 54,
        "end": 64
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 65,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 65,
        "end": 74
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 75,
          "end": 82
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 75,
        "end": 82
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 83,
          "end": 91
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 83,
        "end": 91
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 92,
          "end": 102
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 109,
                "end": 111
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 109,
              "end": 111
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 116,
                "end": 133
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 116,
              "end": 133
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 138,
                "end": 150
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 138,
              "end": 150
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 155,
                "end": 165
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 155,
              "end": 165
            }
          }
        ],
        "loc": {
          "start": 103,
          "end": 167
        }
      },
      "loc": {
        "start": 92,
        "end": 167
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 168,
          "end": 179
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 186,
                "end": 188
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 186,
              "end": 188
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 193,
                "end": 207
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 193,
              "end": 207
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 212,
                "end": 220
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 212,
              "end": 220
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 225,
                "end": 234
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 225,
              "end": 234
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 239,
                "end": 249
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 239,
              "end": 249
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 254,
                "end": 259
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 254,
              "end": 259
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 264,
                "end": 271
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 264,
              "end": 271
            }
          }
        ],
        "loc": {
          "start": 180,
          "end": 273
        }
      },
      "loc": {
        "start": 168,
        "end": 273
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isLoggedIn",
        "loc": {
          "start": 311,
          "end": 321
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 311,
        "end": 321
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timeZone",
        "loc": {
          "start": 322,
          "end": 330
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 322,
        "end": 330
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "users",
        "loc": {
          "start": 331,
          "end": 336
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "activeFocusMode",
              "loc": {
                "start": 343,
                "end": 358
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "mode",
                    "loc": {
                      "start": 369,
                      "end": 373
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "filters",
                          "loc": {
                            "start": 388,
                            "end": 395
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 414,
                                  "end": 416
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 414,
                                "end": 416
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 433,
                                  "end": 443
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 433,
                                "end": 443
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 460,
                                  "end": 463
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 486,
                                        "end": 488
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 486,
                                      "end": 488
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 509,
                                        "end": 519
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 509,
                                      "end": 519
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 540,
                                        "end": 543
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 540,
                                      "end": 543
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 564,
                                        "end": 573
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 564,
                                      "end": 573
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 594,
                                        "end": 606
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 633,
                                              "end": 635
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 633,
                                            "end": 635
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 660,
                                              "end": 668
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 660,
                                            "end": 668
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 693,
                                              "end": 704
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 693,
                                            "end": 704
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 607,
                                        "end": 726
                                      }
                                    },
                                    "loc": {
                                      "start": 594,
                                      "end": 726
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 747,
                                        "end": 750
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isOwn",
                                            "loc": {
                                              "start": 777,
                                              "end": 782
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 777,
                                            "end": 782
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 807,
                                              "end": 819
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 807,
                                            "end": 819
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 751,
                                        "end": 841
                                      }
                                    },
                                    "loc": {
                                      "start": 747,
                                      "end": 841
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 464,
                                  "end": 859
                                }
                              },
                              "loc": {
                                "start": 460,
                                "end": 859
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 876,
                                  "end": 885
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "labels",
                                      "loc": {
                                        "start": 908,
                                        "end": 914
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 941,
                                              "end": 943
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 941,
                                            "end": 943
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 968,
                                              "end": 973
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 968,
                                            "end": 973
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 998,
                                              "end": 1003
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 998,
                                            "end": 1003
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 915,
                                        "end": 1025
                                      }
                                    },
                                    "loc": {
                                      "start": 908,
                                      "end": 1025
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderList",
                                      "loc": {
                                        "start": 1046,
                                        "end": 1058
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1085,
                                              "end": 1087
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1085,
                                            "end": 1087
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 1112,
                                              "end": 1122
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1112,
                                            "end": 1122
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 1147,
                                              "end": 1157
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1147,
                                            "end": 1157
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminders",
                                            "loc": {
                                              "start": 1182,
                                              "end": 1191
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 1222,
                                                    "end": 1224
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1222,
                                                  "end": 1224
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 1253,
                                                    "end": 1263
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1253,
                                                  "end": 1263
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 1292,
                                                    "end": 1302
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1292,
                                                  "end": 1302
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 1331,
                                                    "end": 1335
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1331,
                                                  "end": 1335
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 1364,
                                                    "end": 1375
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1364,
                                                  "end": 1375
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 1404,
                                                    "end": 1411
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1404,
                                                  "end": 1411
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 1440,
                                                    "end": 1445
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1440,
                                                  "end": 1445
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 1474,
                                                    "end": 1484
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1474,
                                                  "end": 1484
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminderItems",
                                                  "loc": {
                                                    "start": 1513,
                                                    "end": 1526
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 1561,
                                                          "end": 1563
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1561,
                                                        "end": 1563
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 1596,
                                                          "end": 1606
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1596,
                                                        "end": 1606
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 1639,
                                                          "end": 1649
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1639,
                                                        "end": 1649
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 1682,
                                                          "end": 1686
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1682,
                                                        "end": 1686
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 1719,
                                                          "end": 1730
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1719,
                                                        "end": 1730
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 1763,
                                                          "end": 1770
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1763,
                                                        "end": 1770
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 1803,
                                                          "end": 1808
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1803,
                                                        "end": 1808
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 1841,
                                                          "end": 1851
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1841,
                                                        "end": 1851
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 1527,
                                                    "end": 1881
                                                  }
                                                },
                                                "loc": {
                                                  "start": 1513,
                                                  "end": 1881
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1192,
                                              "end": 1907
                                            }
                                          },
                                          "loc": {
                                            "start": 1182,
                                            "end": 1907
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1059,
                                        "end": 1929
                                      }
                                    },
                                    "loc": {
                                      "start": 1046,
                                      "end": 1929
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resourceList",
                                      "loc": {
                                        "start": 1950,
                                        "end": 1962
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1989,
                                              "end": 1991
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1989,
                                            "end": 1991
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 2016,
                                              "end": 2026
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2016,
                                            "end": 2026
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 2051,
                                              "end": 2063
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 2094,
                                                    "end": 2096
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2094,
                                                  "end": 2096
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 2125,
                                                    "end": 2133
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2125,
                                                  "end": 2133
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 2162,
                                                    "end": 2173
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2162,
                                                  "end": 2173
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 2202,
                                                    "end": 2206
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2202,
                                                  "end": 2206
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2064,
                                              "end": 2232
                                            }
                                          },
                                          "loc": {
                                            "start": 2051,
                                            "end": 2232
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resources",
                                            "loc": {
                                              "start": 2257,
                                              "end": 2266
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 2297,
                                                    "end": 2299
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2297,
                                                  "end": 2299
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 2328,
                                                    "end": 2333
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2328,
                                                  "end": 2333
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "link",
                                                  "loc": {
                                                    "start": 2362,
                                                    "end": 2366
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2362,
                                                  "end": 2366
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "usedFor",
                                                  "loc": {
                                                    "start": 2395,
                                                    "end": 2402
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2395,
                                                  "end": 2402
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 2431,
                                                    "end": 2443
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 2478,
                                                          "end": 2480
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2478,
                                                        "end": 2480
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 2513,
                                                          "end": 2521
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2513,
                                                        "end": 2521
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 2554,
                                                          "end": 2565
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2554,
                                                        "end": 2565
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 2598,
                                                          "end": 2602
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2598,
                                                        "end": 2602
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2444,
                                                    "end": 2632
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2431,
                                                  "end": 2632
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2267,
                                              "end": 2658
                                            }
                                          },
                                          "loc": {
                                            "start": 2257,
                                            "end": 2658
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1963,
                                        "end": 2680
                                      }
                                    },
                                    "loc": {
                                      "start": 1950,
                                      "end": 2680
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 2701,
                                        "end": 2709
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "FragmentSpread",
                                          "name": {
                                            "kind": "Name",
                                            "value": "Schedule_common",
                                            "loc": {
                                              "start": 2739,
                                              "end": 2754
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 2736,
                                            "end": 2754
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2710,
                                        "end": 2776
                                      }
                                    },
                                    "loc": {
                                      "start": 2701,
                                      "end": 2776
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2797,
                                        "end": 2799
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2797,
                                      "end": 2799
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 2820,
                                        "end": 2824
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2820,
                                      "end": 2824
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 2845,
                                        "end": 2856
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2845,
                                      "end": 2856
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 886,
                                  "end": 2874
                                }
                              },
                              "loc": {
                                "start": 876,
                                "end": 2874
                              }
                            }
                          ],
                          "loc": {
                            "start": 396,
                            "end": 2888
                          }
                        },
                        "loc": {
                          "start": 388,
                          "end": 2888
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 2901,
                            "end": 2907
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2926,
                                  "end": 2928
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2926,
                                "end": 2928
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 2945,
                                  "end": 2950
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2945,
                                "end": 2950
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 2967,
                                  "end": 2972
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2967,
                                "end": 2972
                              }
                            }
                          ],
                          "loc": {
                            "start": 2908,
                            "end": 2986
                          }
                        },
                        "loc": {
                          "start": 2901,
                          "end": 2986
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 2999,
                            "end": 3011
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3030,
                                  "end": 3032
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3030,
                                "end": 3032
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3049,
                                  "end": 3059
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3049,
                                "end": 3059
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3076,
                                  "end": 3086
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3076,
                                "end": 3086
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 3103,
                                  "end": 3112
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3135,
                                        "end": 3137
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3135,
                                      "end": 3137
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3158,
                                        "end": 3168
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3158,
                                      "end": 3168
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3189,
                                        "end": 3199
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3189,
                                      "end": 3199
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3220,
                                        "end": 3224
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3220,
                                      "end": 3224
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 3245,
                                        "end": 3256
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3245,
                                      "end": 3256
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 3277,
                                        "end": 3284
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3277,
                                      "end": 3284
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 3305,
                                        "end": 3310
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3305,
                                      "end": 3310
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 3331,
                                        "end": 3341
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3331,
                                      "end": 3341
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 3362,
                                        "end": 3375
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3402,
                                              "end": 3404
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3402,
                                            "end": 3404
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 3429,
                                              "end": 3439
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3429,
                                            "end": 3439
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 3464,
                                              "end": 3474
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3464,
                                            "end": 3474
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3499,
                                              "end": 3503
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3499,
                                            "end": 3503
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3528,
                                              "end": 3539
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3528,
                                            "end": 3539
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 3564,
                                              "end": 3571
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3564,
                                            "end": 3571
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 3596,
                                              "end": 3601
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3596,
                                            "end": 3601
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 3626,
                                              "end": 3636
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3626,
                                            "end": 3636
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3376,
                                        "end": 3658
                                      }
                                    },
                                    "loc": {
                                      "start": 3362,
                                      "end": 3658
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3113,
                                  "end": 3676
                                }
                              },
                              "loc": {
                                "start": 3103,
                                "end": 3676
                              }
                            }
                          ],
                          "loc": {
                            "start": 3012,
                            "end": 3690
                          }
                        },
                        "loc": {
                          "start": 2999,
                          "end": 3690
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 3703,
                            "end": 3715
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3734,
                                  "end": 3736
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3734,
                                "end": 3736
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3753,
                                  "end": 3763
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3753,
                                "end": 3763
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 3780,
                                  "end": 3792
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3815,
                                        "end": 3817
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3815,
                                      "end": 3817
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 3838,
                                        "end": 3846
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3838,
                                      "end": 3846
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 3867,
                                        "end": 3878
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3867,
                                      "end": 3878
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3899,
                                        "end": 3903
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3899,
                                      "end": 3903
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3793,
                                  "end": 3921
                                }
                              },
                              "loc": {
                                "start": 3780,
                                "end": 3921
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 3938,
                                  "end": 3947
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3970,
                                        "end": 3972
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3970,
                                      "end": 3972
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 3993,
                                        "end": 3998
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3993,
                                      "end": 3998
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 4019,
                                        "end": 4023
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4019,
                                      "end": 4023
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 4044,
                                        "end": 4051
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4044,
                                      "end": 4051
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4072,
                                        "end": 4084
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4111,
                                              "end": 4113
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4111,
                                            "end": 4113
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4138,
                                              "end": 4146
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4138,
                                            "end": 4146
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4171,
                                              "end": 4182
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4171,
                                            "end": 4182
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4207,
                                              "end": 4211
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4207,
                                            "end": 4211
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4085,
                                        "end": 4233
                                      }
                                    },
                                    "loc": {
                                      "start": 4072,
                                      "end": 4233
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3948,
                                  "end": 4251
                                }
                              },
                              "loc": {
                                "start": 3938,
                                "end": 4251
                              }
                            }
                          ],
                          "loc": {
                            "start": 3716,
                            "end": 4265
                          }
                        },
                        "loc": {
                          "start": 3703,
                          "end": 4265
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 4278,
                            "end": 4286
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 4308,
                                  "end": 4323
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 4305,
                                "end": 4323
                              }
                            }
                          ],
                          "loc": {
                            "start": 4287,
                            "end": 4337
                          }
                        },
                        "loc": {
                          "start": 4278,
                          "end": 4337
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4350,
                            "end": 4352
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4350,
                          "end": 4352
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4365,
                            "end": 4369
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4365,
                          "end": 4369
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4382,
                            "end": 4393
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4382,
                          "end": 4393
                        }
                      }
                    ],
                    "loc": {
                      "start": 374,
                      "end": 4403
                    }
                  },
                  "loc": {
                    "start": 369,
                    "end": 4403
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stopCondition",
                    "loc": {
                      "start": 4412,
                      "end": 4425
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4412,
                    "end": 4425
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stopTime",
                    "loc": {
                      "start": 4434,
                      "end": 4442
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4434,
                    "end": 4442
                  }
                }
              ],
              "loc": {
                "start": 359,
                "end": 4448
              }
            },
            "loc": {
              "start": 343,
              "end": 4448
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apisCount",
              "loc": {
                "start": 4453,
                "end": 4462
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4453,
              "end": 4462
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarkLists",
              "loc": {
                "start": 4467,
                "end": 4480
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4491,
                      "end": 4493
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4491,
                    "end": 4493
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4502,
                      "end": 4512
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4502,
                    "end": 4512
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4521,
                      "end": 4531
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4521,
                    "end": 4531
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "label",
                    "loc": {
                      "start": 4540,
                      "end": 4545
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4540,
                    "end": 4545
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarksCount",
                    "loc": {
                      "start": 4554,
                      "end": 4568
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4554,
                    "end": 4568
                  }
                }
              ],
              "loc": {
                "start": 4481,
                "end": 4574
              }
            },
            "loc": {
              "start": 4467,
              "end": 4574
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModes",
              "loc": {
                "start": 4579,
                "end": 4589
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "filters",
                    "loc": {
                      "start": 4600,
                      "end": 4607
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4622,
                            "end": 4624
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4622,
                          "end": 4624
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "filterType",
                          "loc": {
                            "start": 4637,
                            "end": 4647
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4637,
                          "end": 4647
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "tag",
                          "loc": {
                            "start": 4660,
                            "end": 4663
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4682,
                                  "end": 4684
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4682,
                                "end": 4684
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4701,
                                  "end": 4711
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4701,
                                "end": 4711
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 4728,
                                  "end": 4731
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4728,
                                "end": 4731
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bookmarks",
                                "loc": {
                                  "start": 4748,
                                  "end": 4757
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4748,
                                "end": 4757
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 4774,
                                  "end": 4786
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4809,
                                        "end": 4811
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4809,
                                      "end": 4811
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 4832,
                                        "end": 4840
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4832,
                                      "end": 4840
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4861,
                                        "end": 4872
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4861,
                                      "end": 4872
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4787,
                                  "end": 4890
                                }
                              },
                              "loc": {
                                "start": 4774,
                                "end": 4890
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 4907,
                                  "end": 4910
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isOwn",
                                      "loc": {
                                        "start": 4933,
                                        "end": 4938
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4933,
                                      "end": 4938
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 4959,
                                        "end": 4971
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4959,
                                      "end": 4971
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4911,
                                  "end": 4989
                                }
                              },
                              "loc": {
                                "start": 4907,
                                "end": 4989
                              }
                            }
                          ],
                          "loc": {
                            "start": 4664,
                            "end": 5003
                          }
                        },
                        "loc": {
                          "start": 4660,
                          "end": 5003
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "focusMode",
                          "loc": {
                            "start": 5016,
                            "end": 5025
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 5044,
                                  "end": 5050
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 5073,
                                        "end": 5075
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5073,
                                      "end": 5075
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 5096,
                                        "end": 5101
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5096,
                                      "end": 5101
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 5122,
                                        "end": 5127
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5122,
                                      "end": 5127
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5051,
                                  "end": 5145
                                }
                              },
                              "loc": {
                                "start": 5044,
                                "end": 5145
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 5162,
                                  "end": 5174
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 5197,
                                        "end": 5199
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5197,
                                      "end": 5199
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 5220,
                                        "end": 5230
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5220,
                                      "end": 5230
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 5251,
                                        "end": 5261
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5251,
                                      "end": 5261
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 5282,
                                        "end": 5291
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 5318,
                                              "end": 5320
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5318,
                                            "end": 5320
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 5345,
                                              "end": 5355
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5345,
                                            "end": 5355
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 5380,
                                              "end": 5390
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5380,
                                            "end": 5390
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 5415,
                                              "end": 5419
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5415,
                                            "end": 5419
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5444,
                                              "end": 5455
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5444,
                                            "end": 5455
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 5480,
                                              "end": 5487
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5480,
                                            "end": 5487
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 5512,
                                              "end": 5517
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5512,
                                            "end": 5517
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 5542,
                                              "end": 5552
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5542,
                                            "end": 5552
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 5577,
                                              "end": 5590
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 5621,
                                                    "end": 5623
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5621,
                                                  "end": 5623
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 5652,
                                                    "end": 5662
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5652,
                                                  "end": 5662
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 5691,
                                                    "end": 5701
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5691,
                                                  "end": 5701
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 5730,
                                                    "end": 5734
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5730,
                                                  "end": 5734
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 5763,
                                                    "end": 5774
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5763,
                                                  "end": 5774
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 5803,
                                                    "end": 5810
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5803,
                                                  "end": 5810
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 5839,
                                                    "end": 5844
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5839,
                                                  "end": 5844
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 5873,
                                                    "end": 5883
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5873,
                                                  "end": 5883
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 5591,
                                              "end": 5909
                                            }
                                          },
                                          "loc": {
                                            "start": 5577,
                                            "end": 5909
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5292,
                                        "end": 5931
                                      }
                                    },
                                    "loc": {
                                      "start": 5282,
                                      "end": 5931
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5175,
                                  "end": 5949
                                }
                              },
                              "loc": {
                                "start": 5162,
                                "end": 5949
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 5966,
                                  "end": 5978
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 6001,
                                        "end": 6003
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6001,
                                      "end": 6003
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 6024,
                                        "end": 6034
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6024,
                                      "end": 6034
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 6055,
                                        "end": 6067
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 6094,
                                              "end": 6096
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6094,
                                            "end": 6096
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 6121,
                                              "end": 6129
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6121,
                                            "end": 6129
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 6154,
                                              "end": 6165
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6154,
                                            "end": 6165
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 6190,
                                              "end": 6194
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6190,
                                            "end": 6194
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6068,
                                        "end": 6216
                                      }
                                    },
                                    "loc": {
                                      "start": 6055,
                                      "end": 6216
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 6237,
                                        "end": 6246
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 6273,
                                              "end": 6275
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6273,
                                            "end": 6275
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 6300,
                                              "end": 6305
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6300,
                                            "end": 6305
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 6330,
                                              "end": 6334
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6330,
                                            "end": 6334
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 6359,
                                              "end": 6366
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6359,
                                            "end": 6366
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 6391,
                                              "end": 6403
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 6434,
                                                    "end": 6436
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6434,
                                                  "end": 6436
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 6465,
                                                    "end": 6473
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6465,
                                                  "end": 6473
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 6502,
                                                    "end": 6513
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6502,
                                                  "end": 6513
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 6542,
                                                    "end": 6546
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6542,
                                                  "end": 6546
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6404,
                                              "end": 6572
                                            }
                                          },
                                          "loc": {
                                            "start": 6391,
                                            "end": 6572
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6247,
                                        "end": 6594
                                      }
                                    },
                                    "loc": {
                                      "start": 6237,
                                      "end": 6594
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5979,
                                  "end": 6612
                                }
                              },
                              "loc": {
                                "start": 5966,
                                "end": 6612
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 6629,
                                  "end": 6637
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "FragmentSpread",
                                    "name": {
                                      "kind": "Name",
                                      "value": "Schedule_common",
                                      "loc": {
                                        "start": 6663,
                                        "end": 6678
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 6660,
                                      "end": 6678
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 6638,
                                  "end": 6696
                                }
                              },
                              "loc": {
                                "start": 6629,
                                "end": 6696
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 6713,
                                  "end": 6715
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6713,
                                "end": 6715
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 6732,
                                  "end": 6736
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6732,
                                "end": 6736
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 6753,
                                  "end": 6764
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6753,
                                "end": 6764
                              }
                            }
                          ],
                          "loc": {
                            "start": 5026,
                            "end": 6778
                          }
                        },
                        "loc": {
                          "start": 5016,
                          "end": 6778
                        }
                      }
                    ],
                    "loc": {
                      "start": 4608,
                      "end": 6788
                    }
                  },
                  "loc": {
                    "start": 4600,
                    "end": 6788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 6797,
                      "end": 6803
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6818,
                            "end": 6820
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6818,
                          "end": 6820
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 6833,
                            "end": 6838
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6833,
                          "end": 6838
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 6851,
                            "end": 6856
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6851,
                          "end": 6856
                        }
                      }
                    ],
                    "loc": {
                      "start": 6804,
                      "end": 6866
                    }
                  },
                  "loc": {
                    "start": 6797,
                    "end": 6866
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reminderList",
                    "loc": {
                      "start": 6875,
                      "end": 6887
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6902,
                            "end": 6904
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6902,
                          "end": 6904
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 6917,
                            "end": 6927
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6917,
                          "end": 6927
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 6940,
                            "end": 6950
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6940,
                          "end": 6950
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminders",
                          "loc": {
                            "start": 6963,
                            "end": 6972
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 6991,
                                  "end": 6993
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6991,
                                "end": 6993
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 7010,
                                  "end": 7020
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7010,
                                "end": 7020
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 7037,
                                  "end": 7047
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7037,
                                "end": 7047
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 7064,
                                  "end": 7068
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7064,
                                "end": 7068
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 7085,
                                  "end": 7096
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7085,
                                "end": 7096
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "dueDate",
                                "loc": {
                                  "start": 7113,
                                  "end": 7120
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7113,
                                "end": 7120
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 7137,
                                  "end": 7142
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7137,
                                "end": 7142
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isComplete",
                                "loc": {
                                  "start": 7159,
                                  "end": 7169
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7159,
                                "end": 7169
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderItems",
                                "loc": {
                                  "start": 7186,
                                  "end": 7199
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 7222,
                                        "end": 7224
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7222,
                                      "end": 7224
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 7245,
                                        "end": 7255
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7245,
                                      "end": 7255
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 7276,
                                        "end": 7286
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7276,
                                      "end": 7286
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 7307,
                                        "end": 7311
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7307,
                                      "end": 7311
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 7332,
                                        "end": 7343
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7332,
                                      "end": 7343
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 7364,
                                        "end": 7371
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7364,
                                      "end": 7371
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 7392,
                                        "end": 7397
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7392,
                                      "end": 7397
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 7418,
                                        "end": 7428
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7418,
                                      "end": 7428
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 7200,
                                  "end": 7446
                                }
                              },
                              "loc": {
                                "start": 7186,
                                "end": 7446
                              }
                            }
                          ],
                          "loc": {
                            "start": 6973,
                            "end": 7460
                          }
                        },
                        "loc": {
                          "start": 6963,
                          "end": 7460
                        }
                      }
                    ],
                    "loc": {
                      "start": 6888,
                      "end": 7470
                    }
                  },
                  "loc": {
                    "start": 6875,
                    "end": 7470
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resourceList",
                    "loc": {
                      "start": 7479,
                      "end": 7491
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 7506,
                            "end": 7508
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7506,
                          "end": 7508
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 7521,
                            "end": 7531
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7521,
                          "end": 7531
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 7544,
                            "end": 7556
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 7575,
                                  "end": 7577
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7575,
                                "end": 7577
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 7594,
                                  "end": 7602
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7594,
                                "end": 7602
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 7619,
                                  "end": 7630
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7619,
                                "end": 7630
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 7647,
                                  "end": 7651
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7647,
                                "end": 7651
                              }
                            }
                          ],
                          "loc": {
                            "start": 7557,
                            "end": 7665
                          }
                        },
                        "loc": {
                          "start": 7544,
                          "end": 7665
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resources",
                          "loc": {
                            "start": 7678,
                            "end": 7687
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 7706,
                                  "end": 7708
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7706,
                                "end": 7708
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 7725,
                                  "end": 7730
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7725,
                                "end": 7730
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "link",
                                "loc": {
                                  "start": 7747,
                                  "end": 7751
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7747,
                                "end": 7751
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "usedFor",
                                "loc": {
                                  "start": 7768,
                                  "end": 7775
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7768,
                                "end": 7775
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 7792,
                                  "end": 7804
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 7827,
                                        "end": 7829
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7827,
                                      "end": 7829
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 7850,
                                        "end": 7858
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7850,
                                      "end": 7858
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 7879,
                                        "end": 7890
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7879,
                                      "end": 7890
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 7911,
                                        "end": 7915
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7911,
                                      "end": 7915
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 7805,
                                  "end": 7933
                                }
                              },
                              "loc": {
                                "start": 7792,
                                "end": 7933
                              }
                            }
                          ],
                          "loc": {
                            "start": 7688,
                            "end": 7947
                          }
                        },
                        "loc": {
                          "start": 7678,
                          "end": 7947
                        }
                      }
                    ],
                    "loc": {
                      "start": 7492,
                      "end": 7957
                    }
                  },
                  "loc": {
                    "start": 7479,
                    "end": 7957
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "schedule",
                    "loc": {
                      "start": 7966,
                      "end": 7974
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Schedule_common",
                          "loc": {
                            "start": 7992,
                            "end": 8007
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7989,
                          "end": 8007
                        }
                      }
                    ],
                    "loc": {
                      "start": 7975,
                      "end": 8017
                    }
                  },
                  "loc": {
                    "start": 7966,
                    "end": 8017
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 8026,
                      "end": 8028
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8026,
                    "end": 8028
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 8037,
                      "end": 8041
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8037,
                    "end": 8041
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 8050,
                      "end": 8061
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8050,
                    "end": 8061
                  }
                }
              ],
              "loc": {
                "start": 4590,
                "end": 8067
              }
            },
            "loc": {
              "start": 4579,
              "end": 8067
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 8072,
                "end": 8078
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8072,
              "end": 8078
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasPremium",
              "loc": {
                "start": 8083,
                "end": 8093
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8083,
              "end": 8093
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8098,
                "end": 8100
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8098,
              "end": 8100
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "languages",
              "loc": {
                "start": 8105,
                "end": 8114
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8105,
              "end": 8114
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membershipsCount",
              "loc": {
                "start": 8119,
                "end": 8135
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8119,
              "end": 8135
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8140,
                "end": 8144
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8140,
              "end": 8144
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notesCount",
              "loc": {
                "start": 8149,
                "end": 8159
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8149,
              "end": 8159
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectsCount",
              "loc": {
                "start": 8164,
                "end": 8177
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8164,
              "end": 8177
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsAskedCount",
              "loc": {
                "start": 8182,
                "end": 8201
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8182,
              "end": 8201
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routinesCount",
              "loc": {
                "start": 8206,
                "end": 8219
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8206,
              "end": 8219
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractsCount",
              "loc": {
                "start": 8224,
                "end": 8243
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8224,
              "end": 8243
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardsCount",
              "loc": {
                "start": 8248,
                "end": 8262
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8248,
              "end": 8262
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "theme",
              "loc": {
                "start": 8267,
                "end": 8272
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8267,
              "end": 8272
            }
          }
        ],
        "loc": {
          "start": 337,
          "end": 8274
        }
      },
      "loc": {
        "start": 331,
        "end": 8274
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8312,
          "end": 8314
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8312,
        "end": 8314
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 8315,
          "end": 8319
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8315,
        "end": 8319
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "publicAddress",
        "loc": {
          "start": 8320,
          "end": 8333
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8320,
        "end": 8333
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "stakingAddress",
        "loc": {
          "start": 8334,
          "end": 8348
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8334,
        "end": 8348
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "verified",
        "loc": {
          "start": 8349,
          "end": 8357
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8349,
        "end": 8357
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 10,
          "end": 25
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 29,
            "end": 37
          }
        },
        "loc": {
          "start": 29,
          "end": 37
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 40,
                "end": 42
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 40,
              "end": 42
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 43,
                "end": 53
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 43,
              "end": 53
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 54,
                "end": 64
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 54,
              "end": 64
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 65,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 65,
              "end": 74
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 75,
                "end": 82
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 75,
              "end": 82
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 83,
                "end": 91
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 83,
              "end": 91
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 92,
                "end": 102
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 109,
                      "end": 111
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 109,
                    "end": 111
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 116,
                      "end": 133
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 116,
                    "end": 133
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 138,
                      "end": 150
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 138,
                    "end": 150
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 155,
                      "end": 165
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 155,
                    "end": 165
                  }
                }
              ],
              "loc": {
                "start": 103,
                "end": 167
              }
            },
            "loc": {
              "start": 92,
              "end": 167
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 168,
                "end": 179
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 186,
                      "end": 188
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 186,
                    "end": 188
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 193,
                      "end": 207
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 193,
                    "end": 207
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 212,
                      "end": 220
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 212,
                    "end": 220
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 225,
                      "end": 234
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 225,
                    "end": 234
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 239,
                      "end": 249
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 239,
                    "end": 249
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 254,
                      "end": 259
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 254,
                    "end": 259
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 264,
                      "end": 271
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 264,
                    "end": 271
                  }
                }
              ],
              "loc": {
                "start": 180,
                "end": 273
              }
            },
            "loc": {
              "start": 168,
              "end": 273
            }
          }
        ],
        "loc": {
          "start": 38,
          "end": 275
        }
      },
      "loc": {
        "start": 1,
        "end": 275
      }
    },
    "Session_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Session_full",
        "loc": {
          "start": 285,
          "end": 297
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Session",
          "loc": {
            "start": 301,
            "end": 308
          }
        },
        "loc": {
          "start": 301,
          "end": 308
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLoggedIn",
              "loc": {
                "start": 311,
                "end": 321
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 311,
              "end": 321
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeZone",
              "loc": {
                "start": 322,
                "end": 330
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 322,
              "end": 330
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 331,
                "end": 336
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "activeFocusMode",
                    "loc": {
                      "start": 343,
                      "end": 358
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "mode",
                          "loc": {
                            "start": 369,
                            "end": 373
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filters",
                                "loc": {
                                  "start": 388,
                                  "end": 395
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 414,
                                        "end": 416
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 414,
                                      "end": 416
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "filterType",
                                      "loc": {
                                        "start": 433,
                                        "end": 443
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 433,
                                      "end": 443
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 460,
                                        "end": 463
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 486,
                                              "end": 488
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 486,
                                            "end": 488
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 509,
                                              "end": 519
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 509,
                                            "end": 519
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "tag",
                                            "loc": {
                                              "start": 540,
                                              "end": 543
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 540,
                                            "end": 543
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bookmarks",
                                            "loc": {
                                              "start": 564,
                                              "end": 573
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 564,
                                            "end": 573
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 594,
                                              "end": 606
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 633,
                                                    "end": 635
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 633,
                                                  "end": 635
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 660,
                                                    "end": 668
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 660,
                                                  "end": 668
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 693,
                                                    "end": 704
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 693,
                                                  "end": 704
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 607,
                                              "end": 726
                                            }
                                          },
                                          "loc": {
                                            "start": 594,
                                            "end": 726
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 747,
                                              "end": 750
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isOwn",
                                                  "loc": {
                                                    "start": 777,
                                                    "end": 782
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 777,
                                                  "end": 782
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 807,
                                                    "end": 819
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 807,
                                                  "end": 819
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 751,
                                              "end": 841
                                            }
                                          },
                                          "loc": {
                                            "start": 747,
                                            "end": 841
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 464,
                                        "end": 859
                                      }
                                    },
                                    "loc": {
                                      "start": 460,
                                      "end": 859
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "focusMode",
                                      "loc": {
                                        "start": 876,
                                        "end": 885
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "labels",
                                            "loc": {
                                              "start": 908,
                                              "end": 914
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 941,
                                                    "end": 943
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 941,
                                                  "end": 943
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "color",
                                                  "loc": {
                                                    "start": 968,
                                                    "end": 973
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 968,
                                                  "end": 973
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "label",
                                                  "loc": {
                                                    "start": 998,
                                                    "end": 1003
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 998,
                                                  "end": 1003
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 915,
                                              "end": 1025
                                            }
                                          },
                                          "loc": {
                                            "start": 908,
                                            "end": 1025
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderList",
                                            "loc": {
                                              "start": 1046,
                                              "end": 1058
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 1085,
                                                    "end": 1087
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1085,
                                                  "end": 1087
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 1112,
                                                    "end": 1122
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1112,
                                                  "end": 1122
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 1147,
                                                    "end": 1157
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1147,
                                                  "end": 1157
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminders",
                                                  "loc": {
                                                    "start": 1182,
                                                    "end": 1191
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 1222,
                                                          "end": 1224
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1222,
                                                        "end": 1224
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 1253,
                                                          "end": 1263
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1253,
                                                        "end": 1263
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 1292,
                                                          "end": 1302
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1292,
                                                        "end": 1302
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 1331,
                                                          "end": 1335
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1331,
                                                        "end": 1335
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 1364,
                                                          "end": 1375
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1364,
                                                        "end": 1375
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 1404,
                                                          "end": 1411
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1404,
                                                        "end": 1411
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 1440,
                                                          "end": 1445
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1440,
                                                        "end": 1445
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 1474,
                                                          "end": 1484
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1474,
                                                        "end": 1484
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "reminderItems",
                                                        "loc": {
                                                          "start": 1513,
                                                          "end": 1526
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "selectionSet": {
                                                        "kind": "SelectionSet",
                                                        "selections": [
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "id",
                                                              "loc": {
                                                                "start": 1561,
                                                                "end": 1563
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1561,
                                                              "end": 1563
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "created_at",
                                                              "loc": {
                                                                "start": 1596,
                                                                "end": 1606
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1596,
                                                              "end": 1606
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "updated_at",
                                                              "loc": {
                                                                "start": 1639,
                                                                "end": 1649
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1639,
                                                              "end": 1649
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 1682,
                                                                "end": 1686
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1682,
                                                              "end": 1686
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 1719,
                                                                "end": 1730
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1719,
                                                              "end": 1730
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "dueDate",
                                                              "loc": {
                                                                "start": 1763,
                                                                "end": 1770
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1763,
                                                              "end": 1770
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "index",
                                                              "loc": {
                                                                "start": 1803,
                                                                "end": 1808
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1803,
                                                              "end": 1808
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "isComplete",
                                                              "loc": {
                                                                "start": 1841,
                                                                "end": 1851
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1841,
                                                              "end": 1851
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 1527,
                                                          "end": 1881
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 1513,
                                                        "end": 1881
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 1192,
                                                    "end": 1907
                                                  }
                                                },
                                                "loc": {
                                                  "start": 1182,
                                                  "end": 1907
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1059,
                                              "end": 1929
                                            }
                                          },
                                          "loc": {
                                            "start": 1046,
                                            "end": 1929
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resourceList",
                                            "loc": {
                                              "start": 1950,
                                              "end": 1962
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 1989,
                                                    "end": 1991
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1989,
                                                  "end": 1991
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 2016,
                                                    "end": 2026
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2016,
                                                  "end": 2026
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 2051,
                                                    "end": 2063
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 2094,
                                                          "end": 2096
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2094,
                                                        "end": 2096
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 2125,
                                                          "end": 2133
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2125,
                                                        "end": 2133
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 2162,
                                                          "end": 2173
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2162,
                                                        "end": 2173
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 2202,
                                                          "end": 2206
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2202,
                                                        "end": 2206
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2064,
                                                    "end": 2232
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2051,
                                                  "end": 2232
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "resources",
                                                  "loc": {
                                                    "start": 2257,
                                                    "end": 2266
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 2297,
                                                          "end": 2299
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2297,
                                                        "end": 2299
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 2328,
                                                          "end": 2333
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2328,
                                                        "end": 2333
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "link",
                                                        "loc": {
                                                          "start": 2362,
                                                          "end": 2366
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2362,
                                                        "end": 2366
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "usedFor",
                                                        "loc": {
                                                          "start": 2395,
                                                          "end": 2402
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2395,
                                                        "end": 2402
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "translations",
                                                        "loc": {
                                                          "start": 2431,
                                                          "end": 2443
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "selectionSet": {
                                                        "kind": "SelectionSet",
                                                        "selections": [
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "id",
                                                              "loc": {
                                                                "start": 2478,
                                                                "end": 2480
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2478,
                                                              "end": 2480
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "language",
                                                              "loc": {
                                                                "start": 2513,
                                                                "end": 2521
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2513,
                                                              "end": 2521
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 2554,
                                                                "end": 2565
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2554,
                                                              "end": 2565
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 2598,
                                                                "end": 2602
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2598,
                                                              "end": 2602
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 2444,
                                                          "end": 2632
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 2431,
                                                        "end": 2632
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2267,
                                                    "end": 2658
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2257,
                                                  "end": 2658
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1963,
                                              "end": 2680
                                            }
                                          },
                                          "loc": {
                                            "start": 1950,
                                            "end": 2680
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "schedule",
                                            "loc": {
                                              "start": 2701,
                                              "end": 2709
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "FragmentSpread",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "Schedule_common",
                                                  "loc": {
                                                    "start": 2739,
                                                    "end": 2754
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 2736,
                                                  "end": 2754
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2710,
                                              "end": 2776
                                            }
                                          },
                                          "loc": {
                                            "start": 2701,
                                            "end": 2776
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 2797,
                                              "end": 2799
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2797,
                                            "end": 2799
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 2820,
                                              "end": 2824
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2820,
                                            "end": 2824
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 2845,
                                              "end": 2856
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2845,
                                            "end": 2856
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 886,
                                        "end": 2874
                                      }
                                    },
                                    "loc": {
                                      "start": 876,
                                      "end": 2874
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 396,
                                  "end": 2888
                                }
                              },
                              "loc": {
                                "start": 388,
                                "end": 2888
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 2901,
                                  "end": 2907
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2926,
                                        "end": 2928
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2926,
                                      "end": 2928
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 2945,
                                        "end": 2950
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2945,
                                      "end": 2950
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 2967,
                                        "end": 2972
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2967,
                                      "end": 2972
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2908,
                                  "end": 2986
                                }
                              },
                              "loc": {
                                "start": 2901,
                                "end": 2986
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 2999,
                                  "end": 3011
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3030,
                                        "end": 3032
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3030,
                                      "end": 3032
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3049,
                                        "end": 3059
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3049,
                                      "end": 3059
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3076,
                                        "end": 3086
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3076,
                                      "end": 3086
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 3103,
                                        "end": 3112
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3135,
                                              "end": 3137
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3135,
                                            "end": 3137
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 3158,
                                              "end": 3168
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3158,
                                            "end": 3168
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 3189,
                                              "end": 3199
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3189,
                                            "end": 3199
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3220,
                                              "end": 3224
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3220,
                                            "end": 3224
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3245,
                                              "end": 3256
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3245,
                                            "end": 3256
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 3277,
                                              "end": 3284
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3277,
                                            "end": 3284
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 3305,
                                              "end": 3310
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3305,
                                            "end": 3310
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 3331,
                                              "end": 3341
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3331,
                                            "end": 3341
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 3362,
                                              "end": 3375
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 3402,
                                                    "end": 3404
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3402,
                                                  "end": 3404
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 3429,
                                                    "end": 3439
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3429,
                                                  "end": 3439
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 3464,
                                                    "end": 3474
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3464,
                                                  "end": 3474
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 3499,
                                                    "end": 3503
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3499,
                                                  "end": 3503
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 3528,
                                                    "end": 3539
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3528,
                                                  "end": 3539
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 3564,
                                                    "end": 3571
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3564,
                                                  "end": 3571
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 3596,
                                                    "end": 3601
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3596,
                                                  "end": 3601
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 3626,
                                                    "end": 3636
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3626,
                                                  "end": 3636
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3376,
                                              "end": 3658
                                            }
                                          },
                                          "loc": {
                                            "start": 3362,
                                            "end": 3658
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3113,
                                        "end": 3676
                                      }
                                    },
                                    "loc": {
                                      "start": 3103,
                                      "end": 3676
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3012,
                                  "end": 3690
                                }
                              },
                              "loc": {
                                "start": 2999,
                                "end": 3690
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 3703,
                                  "end": 3715
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3734,
                                        "end": 3736
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3734,
                                      "end": 3736
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3753,
                                        "end": 3763
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3753,
                                      "end": 3763
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 3780,
                                        "end": 3792
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3815,
                                              "end": 3817
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3815,
                                            "end": 3817
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 3838,
                                              "end": 3846
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3838,
                                            "end": 3846
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3867,
                                              "end": 3878
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3867,
                                            "end": 3878
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3899,
                                              "end": 3903
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3899,
                                            "end": 3903
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3793,
                                        "end": 3921
                                      }
                                    },
                                    "loc": {
                                      "start": 3780,
                                      "end": 3921
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 3938,
                                        "end": 3947
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3970,
                                              "end": 3972
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3970,
                                            "end": 3972
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 3993,
                                              "end": 3998
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3993,
                                            "end": 3998
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 4019,
                                              "end": 4023
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4019,
                                            "end": 4023
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 4044,
                                              "end": 4051
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4044,
                                            "end": 4051
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 4072,
                                              "end": 4084
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 4111,
                                                    "end": 4113
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4111,
                                                  "end": 4113
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 4138,
                                                    "end": 4146
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4138,
                                                  "end": 4146
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 4171,
                                                    "end": 4182
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4171,
                                                  "end": 4182
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 4207,
                                                    "end": 4211
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4207,
                                                  "end": 4211
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 4085,
                                              "end": 4233
                                            }
                                          },
                                          "loc": {
                                            "start": 4072,
                                            "end": 4233
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3948,
                                        "end": 4251
                                      }
                                    },
                                    "loc": {
                                      "start": 3938,
                                      "end": 4251
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3716,
                                  "end": 4265
                                }
                              },
                              "loc": {
                                "start": 3703,
                                "end": 4265
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 4278,
                                  "end": 4286
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "FragmentSpread",
                                    "name": {
                                      "kind": "Name",
                                      "value": "Schedule_common",
                                      "loc": {
                                        "start": 4308,
                                        "end": 4323
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 4305,
                                      "end": 4323
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4287,
                                  "end": 4337
                                }
                              },
                              "loc": {
                                "start": 4278,
                                "end": 4337
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4350,
                                  "end": 4352
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4350,
                                "end": 4352
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 4365,
                                  "end": 4369
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4365,
                                "end": 4369
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 4382,
                                  "end": 4393
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4382,
                                "end": 4393
                              }
                            }
                          ],
                          "loc": {
                            "start": 374,
                            "end": 4403
                          }
                        },
                        "loc": {
                          "start": 369,
                          "end": 4403
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopCondition",
                          "loc": {
                            "start": 4412,
                            "end": 4425
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4412,
                          "end": 4425
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopTime",
                          "loc": {
                            "start": 4434,
                            "end": 4442
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4434,
                          "end": 4442
                        }
                      }
                    ],
                    "loc": {
                      "start": 359,
                      "end": 4448
                    }
                  },
                  "loc": {
                    "start": 343,
                    "end": 4448
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apisCount",
                    "loc": {
                      "start": 4453,
                      "end": 4462
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4453,
                    "end": 4462
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarkLists",
                    "loc": {
                      "start": 4467,
                      "end": 4480
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4491,
                            "end": 4493
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4491,
                          "end": 4493
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 4502,
                            "end": 4512
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4502,
                          "end": 4512
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 4521,
                            "end": 4531
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4521,
                          "end": 4531
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 4540,
                            "end": 4545
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4540,
                          "end": 4545
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bookmarksCount",
                          "loc": {
                            "start": 4554,
                            "end": 4568
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4554,
                          "end": 4568
                        }
                      }
                    ],
                    "loc": {
                      "start": 4481,
                      "end": 4574
                    }
                  },
                  "loc": {
                    "start": 4467,
                    "end": 4574
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusModes",
                    "loc": {
                      "start": 4579,
                      "end": 4589
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "filters",
                          "loc": {
                            "start": 4600,
                            "end": 4607
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4622,
                                  "end": 4624
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4622,
                                "end": 4624
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 4637,
                                  "end": 4647
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4637,
                                "end": 4647
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 4660,
                                  "end": 4663
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4682,
                                        "end": 4684
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4682,
                                      "end": 4684
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4701,
                                        "end": 4711
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4701,
                                      "end": 4711
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 4728,
                                        "end": 4731
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4728,
                                      "end": 4731
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 4748,
                                        "end": 4757
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4748,
                                      "end": 4757
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4774,
                                        "end": 4786
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4809,
                                              "end": 4811
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4809,
                                            "end": 4811
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4832,
                                              "end": 4840
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4832,
                                            "end": 4840
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4861,
                                              "end": 4872
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4861,
                                            "end": 4872
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4787,
                                        "end": 4890
                                      }
                                    },
                                    "loc": {
                                      "start": 4774,
                                      "end": 4890
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 4907,
                                        "end": 4910
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isOwn",
                                            "loc": {
                                              "start": 4933,
                                              "end": 4938
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4933,
                                            "end": 4938
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 4959,
                                              "end": 4971
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4959,
                                            "end": 4971
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4911,
                                        "end": 4989
                                      }
                                    },
                                    "loc": {
                                      "start": 4907,
                                      "end": 4989
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4664,
                                  "end": 5003
                                }
                              },
                              "loc": {
                                "start": 4660,
                                "end": 5003
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 5016,
                                  "end": 5025
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "labels",
                                      "loc": {
                                        "start": 5044,
                                        "end": 5050
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 5073,
                                              "end": 5075
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5073,
                                            "end": 5075
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 5096,
                                              "end": 5101
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5096,
                                            "end": 5101
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 5122,
                                              "end": 5127
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5122,
                                            "end": 5127
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5051,
                                        "end": 5145
                                      }
                                    },
                                    "loc": {
                                      "start": 5044,
                                      "end": 5145
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderList",
                                      "loc": {
                                        "start": 5162,
                                        "end": 5174
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 5197,
                                              "end": 5199
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5197,
                                            "end": 5199
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 5220,
                                              "end": 5230
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5220,
                                            "end": 5230
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 5251,
                                              "end": 5261
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5251,
                                            "end": 5261
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminders",
                                            "loc": {
                                              "start": 5282,
                                              "end": 5291
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 5318,
                                                    "end": 5320
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5318,
                                                  "end": 5320
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 5345,
                                                    "end": 5355
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5345,
                                                  "end": 5355
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 5380,
                                                    "end": 5390
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5380,
                                                  "end": 5390
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 5415,
                                                    "end": 5419
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5415,
                                                  "end": 5419
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 5444,
                                                    "end": 5455
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5444,
                                                  "end": 5455
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 5480,
                                                    "end": 5487
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5480,
                                                  "end": 5487
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 5512,
                                                    "end": 5517
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5512,
                                                  "end": 5517
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 5542,
                                                    "end": 5552
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5542,
                                                  "end": 5552
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminderItems",
                                                  "loc": {
                                                    "start": 5577,
                                                    "end": 5590
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 5621,
                                                          "end": 5623
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5621,
                                                        "end": 5623
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 5652,
                                                          "end": 5662
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5652,
                                                        "end": 5662
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 5691,
                                                          "end": 5701
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5691,
                                                        "end": 5701
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 5730,
                                                          "end": 5734
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5730,
                                                        "end": 5734
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 5763,
                                                          "end": 5774
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5763,
                                                        "end": 5774
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 5803,
                                                          "end": 5810
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5803,
                                                        "end": 5810
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 5839,
                                                          "end": 5844
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5839,
                                                        "end": 5844
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 5873,
                                                          "end": 5883
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5873,
                                                        "end": 5883
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 5591,
                                                    "end": 5909
                                                  }
                                                },
                                                "loc": {
                                                  "start": 5577,
                                                  "end": 5909
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 5292,
                                              "end": 5931
                                            }
                                          },
                                          "loc": {
                                            "start": 5282,
                                            "end": 5931
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5175,
                                        "end": 5949
                                      }
                                    },
                                    "loc": {
                                      "start": 5162,
                                      "end": 5949
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resourceList",
                                      "loc": {
                                        "start": 5966,
                                        "end": 5978
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 6001,
                                              "end": 6003
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6001,
                                            "end": 6003
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 6024,
                                              "end": 6034
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6024,
                                            "end": 6034
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 6055,
                                              "end": 6067
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 6094,
                                                    "end": 6096
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6094,
                                                  "end": 6096
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 6121,
                                                    "end": 6129
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6121,
                                                  "end": 6129
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 6154,
                                                    "end": 6165
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6154,
                                                  "end": 6165
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 6190,
                                                    "end": 6194
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6190,
                                                  "end": 6194
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6068,
                                              "end": 6216
                                            }
                                          },
                                          "loc": {
                                            "start": 6055,
                                            "end": 6216
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resources",
                                            "loc": {
                                              "start": 6237,
                                              "end": 6246
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 6273,
                                                    "end": 6275
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6273,
                                                  "end": 6275
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 6300,
                                                    "end": 6305
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6300,
                                                  "end": 6305
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "link",
                                                  "loc": {
                                                    "start": 6330,
                                                    "end": 6334
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6330,
                                                  "end": 6334
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "usedFor",
                                                  "loc": {
                                                    "start": 6359,
                                                    "end": 6366
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6359,
                                                  "end": 6366
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 6391,
                                                    "end": 6403
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 6434,
                                                          "end": 6436
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6434,
                                                        "end": 6436
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 6465,
                                                          "end": 6473
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6465,
                                                        "end": 6473
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 6502,
                                                          "end": 6513
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6502,
                                                        "end": 6513
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 6542,
                                                          "end": 6546
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6542,
                                                        "end": 6546
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 6404,
                                                    "end": 6572
                                                  }
                                                },
                                                "loc": {
                                                  "start": 6391,
                                                  "end": 6572
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6247,
                                              "end": 6594
                                            }
                                          },
                                          "loc": {
                                            "start": 6237,
                                            "end": 6594
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5979,
                                        "end": 6612
                                      }
                                    },
                                    "loc": {
                                      "start": 5966,
                                      "end": 6612
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 6629,
                                        "end": 6637
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "FragmentSpread",
                                          "name": {
                                            "kind": "Name",
                                            "value": "Schedule_common",
                                            "loc": {
                                              "start": 6663,
                                              "end": 6678
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 6660,
                                            "end": 6678
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6638,
                                        "end": 6696
                                      }
                                    },
                                    "loc": {
                                      "start": 6629,
                                      "end": 6696
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 6713,
                                        "end": 6715
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6713,
                                      "end": 6715
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 6732,
                                        "end": 6736
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6732,
                                      "end": 6736
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 6753,
                                        "end": 6764
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6753,
                                      "end": 6764
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5026,
                                  "end": 6778
                                }
                              },
                              "loc": {
                                "start": 5016,
                                "end": 6778
                              }
                            }
                          ],
                          "loc": {
                            "start": 4608,
                            "end": 6788
                          }
                        },
                        "loc": {
                          "start": 4600,
                          "end": 6788
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 6797,
                            "end": 6803
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 6818,
                                  "end": 6820
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6818,
                                "end": 6820
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 6833,
                                  "end": 6838
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6833,
                                "end": 6838
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 6851,
                                  "end": 6856
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6851,
                                "end": 6856
                              }
                            }
                          ],
                          "loc": {
                            "start": 6804,
                            "end": 6866
                          }
                        },
                        "loc": {
                          "start": 6797,
                          "end": 6866
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 6875,
                            "end": 6887
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 6902,
                                  "end": 6904
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6902,
                                "end": 6904
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 6917,
                                  "end": 6927
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6917,
                                "end": 6927
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 6940,
                                  "end": 6950
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6940,
                                "end": 6950
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 6963,
                                  "end": 6972
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 6991,
                                        "end": 6993
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6991,
                                      "end": 6993
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 7010,
                                        "end": 7020
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7010,
                                      "end": 7020
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 7037,
                                        "end": 7047
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7037,
                                      "end": 7047
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 7064,
                                        "end": 7068
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7064,
                                      "end": 7068
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 7085,
                                        "end": 7096
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7085,
                                      "end": 7096
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 7113,
                                        "end": 7120
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7113,
                                      "end": 7120
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 7137,
                                        "end": 7142
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7137,
                                      "end": 7142
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 7159,
                                        "end": 7169
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7159,
                                      "end": 7169
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 7186,
                                        "end": 7199
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 7222,
                                              "end": 7224
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7222,
                                            "end": 7224
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 7245,
                                              "end": 7255
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7245,
                                            "end": 7255
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 7276,
                                              "end": 7286
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7276,
                                            "end": 7286
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 7307,
                                              "end": 7311
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7307,
                                            "end": 7311
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 7332,
                                              "end": 7343
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7332,
                                            "end": 7343
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 7364,
                                              "end": 7371
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7364,
                                            "end": 7371
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 7392,
                                              "end": 7397
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7392,
                                            "end": 7397
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 7418,
                                              "end": 7428
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7418,
                                            "end": 7428
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 7200,
                                        "end": 7446
                                      }
                                    },
                                    "loc": {
                                      "start": 7186,
                                      "end": 7446
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 6973,
                                  "end": 7460
                                }
                              },
                              "loc": {
                                "start": 6963,
                                "end": 7460
                              }
                            }
                          ],
                          "loc": {
                            "start": 6888,
                            "end": 7470
                          }
                        },
                        "loc": {
                          "start": 6875,
                          "end": 7470
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 7479,
                            "end": 7491
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 7506,
                                  "end": 7508
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7506,
                                "end": 7508
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 7521,
                                  "end": 7531
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7521,
                                "end": 7531
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 7544,
                                  "end": 7556
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 7575,
                                        "end": 7577
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7575,
                                      "end": 7577
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 7594,
                                        "end": 7602
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7594,
                                      "end": 7602
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 7619,
                                        "end": 7630
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7619,
                                      "end": 7630
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 7647,
                                        "end": 7651
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7647,
                                      "end": 7651
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 7557,
                                  "end": 7665
                                }
                              },
                              "loc": {
                                "start": 7544,
                                "end": 7665
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 7678,
                                  "end": 7687
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 7706,
                                        "end": 7708
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7706,
                                      "end": 7708
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 7725,
                                        "end": 7730
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7725,
                                      "end": 7730
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 7747,
                                        "end": 7751
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7747,
                                      "end": 7751
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 7768,
                                        "end": 7775
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7768,
                                      "end": 7775
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 7792,
                                        "end": 7804
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 7827,
                                              "end": 7829
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7827,
                                            "end": 7829
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 7850,
                                              "end": 7858
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7850,
                                            "end": 7858
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 7879,
                                              "end": 7890
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7879,
                                            "end": 7890
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 7911,
                                              "end": 7915
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7911,
                                            "end": 7915
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 7805,
                                        "end": 7933
                                      }
                                    },
                                    "loc": {
                                      "start": 7792,
                                      "end": 7933
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 7688,
                                  "end": 7947
                                }
                              },
                              "loc": {
                                "start": 7678,
                                "end": 7947
                              }
                            }
                          ],
                          "loc": {
                            "start": 7492,
                            "end": 7957
                          }
                        },
                        "loc": {
                          "start": 7479,
                          "end": 7957
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 7966,
                            "end": 7974
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 7992,
                                  "end": 8007
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 7989,
                                "end": 8007
                              }
                            }
                          ],
                          "loc": {
                            "start": 7975,
                            "end": 8017
                          }
                        },
                        "loc": {
                          "start": 7966,
                          "end": 8017
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 8026,
                            "end": 8028
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8026,
                          "end": 8028
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 8037,
                            "end": 8041
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8037,
                          "end": 8041
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 8050,
                            "end": 8061
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8050,
                          "end": 8061
                        }
                      }
                    ],
                    "loc": {
                      "start": 4590,
                      "end": 8067
                    }
                  },
                  "loc": {
                    "start": 4579,
                    "end": 8067
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 8072,
                      "end": 8078
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8072,
                    "end": 8078
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 8083,
                      "end": 8093
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8083,
                    "end": 8093
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 8098,
                      "end": 8100
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8098,
                    "end": 8100
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "languages",
                    "loc": {
                      "start": 8105,
                      "end": 8114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8105,
                    "end": 8114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 8119,
                      "end": 8135
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8119,
                    "end": 8135
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 8140,
                      "end": 8144
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8140,
                    "end": 8144
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 8149,
                      "end": 8159
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8149,
                    "end": 8159
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 8164,
                      "end": 8177
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8164,
                    "end": 8177
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 8182,
                      "end": 8201
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8182,
                    "end": 8201
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 8206,
                      "end": 8219
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8206,
                    "end": 8219
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractsCount",
                    "loc": {
                      "start": 8224,
                      "end": 8243
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8224,
                    "end": 8243
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 8248,
                      "end": 8262
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8248,
                    "end": 8262
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 8267,
                      "end": 8272
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8267,
                    "end": 8272
                  }
                }
              ],
              "loc": {
                "start": 337,
                "end": 8274
              }
            },
            "loc": {
              "start": 331,
              "end": 8274
            }
          }
        ],
        "loc": {
          "start": 309,
          "end": 8276
        }
      },
      "loc": {
        "start": 276,
        "end": 8276
      }
    },
    "Wallet_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Wallet_common",
        "loc": {
          "start": 8286,
          "end": 8299
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Wallet",
          "loc": {
            "start": 8303,
            "end": 8309
          }
        },
        "loc": {
          "start": 8303,
          "end": 8309
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8312,
                "end": 8314
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8312,
              "end": 8314
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8315,
                "end": 8319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8315,
              "end": 8319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "publicAddress",
              "loc": {
                "start": 8320,
                "end": 8333
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8320,
              "end": 8333
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stakingAddress",
              "loc": {
                "start": 8334,
                "end": 8348
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8334,
              "end": 8348
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "verified",
              "loc": {
                "start": 8349,
                "end": 8357
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8349,
              "end": 8357
            }
          }
        ],
        "loc": {
          "start": 8310,
          "end": 8359
        }
      },
      "loc": {
        "start": 8277,
        "end": 8359
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "walletComplete",
      "loc": {
        "start": 8370,
        "end": 8384
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8386,
              "end": 8391
            }
          },
          "loc": {
            "start": 8385,
            "end": 8391
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "WalletCompleteInput",
              "loc": {
                "start": 8393,
                "end": 8412
              }
            },
            "loc": {
              "start": 8393,
              "end": 8412
            }
          },
          "loc": {
            "start": 8393,
            "end": 8413
          }
        },
        "directives": [],
        "loc": {
          "start": 8385,
          "end": 8413
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "walletComplete",
            "loc": {
              "start": 8419,
              "end": 8433
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 8434,
                  "end": 8439
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 8442,
                    "end": 8447
                  }
                },
                "loc": {
                  "start": 8441,
                  "end": 8447
                }
              },
              "loc": {
                "start": 8434,
                "end": 8447
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "firstLogIn",
                  "loc": {
                    "start": 8455,
                    "end": 8465
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 8455,
                  "end": 8465
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "session",
                  "loc": {
                    "start": 8470,
                    "end": 8477
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Session_full",
                        "loc": {
                          "start": 8491,
                          "end": 8503
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8488,
                        "end": 8503
                      }
                    }
                  ],
                  "loc": {
                    "start": 8478,
                    "end": 8509
                  }
                },
                "loc": {
                  "start": 8470,
                  "end": 8509
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "wallet",
                  "loc": {
                    "start": 8514,
                    "end": 8520
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Wallet_common",
                        "loc": {
                          "start": 8534,
                          "end": 8547
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8531,
                        "end": 8547
                      }
                    }
                  ],
                  "loc": {
                    "start": 8521,
                    "end": 8553
                  }
                },
                "loc": {
                  "start": 8514,
                  "end": 8553
                }
              }
            ],
            "loc": {
              "start": 8449,
              "end": 8557
            }
          },
          "loc": {
            "start": 8419,
            "end": 8557
          }
        }
      ],
      "loc": {
        "start": 8415,
        "end": 8559
      }
    },
    "loc": {
      "start": 8361,
      "end": 8559
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_walletComplete"
  }
} as const;
