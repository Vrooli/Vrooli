export const auth_logOut = {
  "fieldName": "logOut",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "logOut",
        "loc": {
          "start": 319,
          "end": 325
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 326,
              "end": 331
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 334,
                "end": 339
              }
            },
            "loc": {
              "start": 333,
              "end": 339
            }
          },
          "loc": {
            "start": 326,
            "end": 339
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
              "value": "isLoggedIn",
              "loc": {
                "start": 347,
                "end": 357
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 347,
              "end": 357
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeZone",
              "loc": {
                "start": 362,
                "end": 370
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 362,
              "end": 370
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 375,
                "end": 380
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
                      "start": 391,
                      "end": 406
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
                            "start": 421,
                            "end": 425
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
                                  "start": 444,
                                  "end": 451
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
                                        "start": 474,
                                        "end": 476
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 474,
                                      "end": 476
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "filterType",
                                      "loc": {
                                        "start": 497,
                                        "end": 507
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 497,
                                      "end": 507
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 528,
                                        "end": 531
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
                                              "start": 558,
                                              "end": 560
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 558,
                                            "end": 560
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 585,
                                              "end": 595
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 585,
                                            "end": 595
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "tag",
                                            "loc": {
                                              "start": 620,
                                              "end": 623
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 620,
                                            "end": 623
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bookmarks",
                                            "loc": {
                                              "start": 648,
                                              "end": 657
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 648,
                                            "end": 657
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 682,
                                              "end": 694
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
                                                    "start": 725,
                                                    "end": 727
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 725,
                                                  "end": 727
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 756,
                                                    "end": 764
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 756,
                                                  "end": 764
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 793,
                                                    "end": 804
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 793,
                                                  "end": 804
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 695,
                                              "end": 830
                                            }
                                          },
                                          "loc": {
                                            "start": 682,
                                            "end": 830
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 855,
                                              "end": 858
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
                                                    "start": 889,
                                                    "end": 894
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 889,
                                                  "end": 894
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 923,
                                                    "end": 935
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 923,
                                                  "end": 935
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 859,
                                              "end": 961
                                            }
                                          },
                                          "loc": {
                                            "start": 855,
                                            "end": 961
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 532,
                                        "end": 983
                                      }
                                    },
                                    "loc": {
                                      "start": 528,
                                      "end": 983
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "focusMode",
                                      "loc": {
                                        "start": 1004,
                                        "end": 1013
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
                                              "start": 1040,
                                              "end": 1046
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
                                                    "start": 1077,
                                                    "end": 1079
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1077,
                                                  "end": 1079
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "color",
                                                  "loc": {
                                                    "start": 1108,
                                                    "end": 1113
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1108,
                                                  "end": 1113
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "label",
                                                  "loc": {
                                                    "start": 1142,
                                                    "end": 1147
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1142,
                                                  "end": 1147
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1047,
                                              "end": 1173
                                            }
                                          },
                                          "loc": {
                                            "start": 1040,
                                            "end": 1173
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderList",
                                            "loc": {
                                              "start": 1198,
                                              "end": 1210
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
                                                    "start": 1241,
                                                    "end": 1243
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1241,
                                                  "end": 1243
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 1272,
                                                    "end": 1282
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1272,
                                                  "end": 1282
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 1311,
                                                    "end": 1321
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1311,
                                                  "end": 1321
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminders",
                                                  "loc": {
                                                    "start": 1350,
                                                    "end": 1359
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
                                                          "start": 1394,
                                                          "end": 1396
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1394,
                                                        "end": 1396
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 1429,
                                                          "end": 1439
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1429,
                                                        "end": 1439
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 1472,
                                                          "end": 1482
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1472,
                                                        "end": 1482
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 1515,
                                                          "end": 1519
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1515,
                                                        "end": 1519
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 1552,
                                                          "end": 1563
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1552,
                                                        "end": 1563
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 1596,
                                                          "end": 1603
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1596,
                                                        "end": 1603
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 1636,
                                                          "end": 1641
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1636,
                                                        "end": 1641
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 1674,
                                                          "end": 1684
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1674,
                                                        "end": 1684
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "reminderItems",
                                                        "loc": {
                                                          "start": 1717,
                                                          "end": 1730
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
                                                                "start": 1769,
                                                                "end": 1771
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1769,
                                                              "end": 1771
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "created_at",
                                                              "loc": {
                                                                "start": 1808,
                                                                "end": 1818
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1808,
                                                              "end": 1818
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "updated_at",
                                                              "loc": {
                                                                "start": 1855,
                                                                "end": 1865
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1855,
                                                              "end": 1865
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 1902,
                                                                "end": 1906
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1902,
                                                              "end": 1906
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 1943,
                                                                "end": 1954
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1943,
                                                              "end": 1954
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "dueDate",
                                                              "loc": {
                                                                "start": 1991,
                                                                "end": 1998
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1991,
                                                              "end": 1998
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "index",
                                                              "loc": {
                                                                "start": 2035,
                                                                "end": 2040
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2035,
                                                              "end": 2040
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "isComplete",
                                                              "loc": {
                                                                "start": 2077,
                                                                "end": 2087
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2077,
                                                              "end": 2087
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 1731,
                                                          "end": 2121
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 1717,
                                                        "end": 2121
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 1360,
                                                    "end": 2151
                                                  }
                                                },
                                                "loc": {
                                                  "start": 1350,
                                                  "end": 2151
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1211,
                                              "end": 2177
                                            }
                                          },
                                          "loc": {
                                            "start": 1198,
                                            "end": 2177
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resourceList",
                                            "loc": {
                                              "start": 2202,
                                              "end": 2214
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
                                                    "start": 2245,
                                                    "end": 2247
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2245,
                                                  "end": 2247
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 2276,
                                                    "end": 2286
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2276,
                                                  "end": 2286
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 2315,
                                                    "end": 2327
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
                                                          "start": 2362,
                                                          "end": 2364
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2362,
                                                        "end": 2364
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 2397,
                                                          "end": 2405
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2397,
                                                        "end": 2405
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 2438,
                                                          "end": 2449
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2438,
                                                        "end": 2449
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 2482,
                                                          "end": 2486
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2482,
                                                        "end": 2486
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2328,
                                                    "end": 2516
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2315,
                                                  "end": 2516
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "resources",
                                                  "loc": {
                                                    "start": 2545,
                                                    "end": 2554
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
                                                          "start": 2589,
                                                          "end": 2591
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2589,
                                                        "end": 2591
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 2624,
                                                          "end": 2629
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2624,
                                                        "end": 2629
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "link",
                                                        "loc": {
                                                          "start": 2662,
                                                          "end": 2666
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2662,
                                                        "end": 2666
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "usedFor",
                                                        "loc": {
                                                          "start": 2699,
                                                          "end": 2706
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2699,
                                                        "end": 2706
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "translations",
                                                        "loc": {
                                                          "start": 2739,
                                                          "end": 2751
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
                                                                "start": 2790,
                                                                "end": 2792
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2790,
                                                              "end": 2792
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "language",
                                                              "loc": {
                                                                "start": 2829,
                                                                "end": 2837
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2829,
                                                              "end": 2837
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 2874,
                                                                "end": 2885
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2874,
                                                              "end": 2885
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 2922,
                                                                "end": 2926
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2922,
                                                              "end": 2926
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 2752,
                                                          "end": 2960
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 2739,
                                                        "end": 2960
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2555,
                                                    "end": 2990
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2545,
                                                  "end": 2990
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2215,
                                              "end": 3016
                                            }
                                          },
                                          "loc": {
                                            "start": 2202,
                                            "end": 3016
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "schedule",
                                            "loc": {
                                              "start": 3041,
                                              "end": 3049
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
                                                    "start": 3083,
                                                    "end": 3098
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 3080,
                                                  "end": 3098
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3050,
                                              "end": 3124
                                            }
                                          },
                                          "loc": {
                                            "start": 3041,
                                            "end": 3124
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3149,
                                              "end": 3151
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3149,
                                            "end": 3151
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3176,
                                              "end": 3180
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3176,
                                            "end": 3180
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3205,
                                              "end": 3216
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3205,
                                            "end": 3216
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1014,
                                        "end": 3238
                                      }
                                    },
                                    "loc": {
                                      "start": 1004,
                                      "end": 3238
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 452,
                                  "end": 3256
                                }
                              },
                              "loc": {
                                "start": 444,
                                "end": 3256
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 3273,
                                  "end": 3279
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
                                        "start": 3302,
                                        "end": 3304
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3302,
                                      "end": 3304
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 3325,
                                        "end": 3330
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3325,
                                      "end": 3330
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 3351,
                                        "end": 3356
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3351,
                                      "end": 3356
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3280,
                                  "end": 3374
                                }
                              },
                              "loc": {
                                "start": 3273,
                                "end": 3374
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 3391,
                                  "end": 3403
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
                                        "start": 3426,
                                        "end": 3428
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3426,
                                      "end": 3428
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3449,
                                        "end": 3459
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3449,
                                      "end": 3459
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3480,
                                        "end": 3490
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3480,
                                      "end": 3490
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 3511,
                                        "end": 3520
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
                                              "start": 3547,
                                              "end": 3549
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3547,
                                            "end": 3549
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 3574,
                                              "end": 3584
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3574,
                                            "end": 3584
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 3609,
                                              "end": 3619
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3609,
                                            "end": 3619
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3644,
                                              "end": 3648
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3644,
                                            "end": 3648
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3673,
                                              "end": 3684
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3673,
                                            "end": 3684
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 3709,
                                              "end": 3716
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3709,
                                            "end": 3716
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 3741,
                                              "end": 3746
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3741,
                                            "end": 3746
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 3771,
                                              "end": 3781
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3771,
                                            "end": 3781
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 3806,
                                              "end": 3819
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
                                                    "start": 3850,
                                                    "end": 3852
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3850,
                                                  "end": 3852
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 3881,
                                                    "end": 3891
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3881,
                                                  "end": 3891
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 3920,
                                                    "end": 3930
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3920,
                                                  "end": 3930
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 3959,
                                                    "end": 3963
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3959,
                                                  "end": 3963
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 3992,
                                                    "end": 4003
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3992,
                                                  "end": 4003
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 4032,
                                                    "end": 4039
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4032,
                                                  "end": 4039
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 4068,
                                                    "end": 4073
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4068,
                                                  "end": 4073
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 4102,
                                                    "end": 4112
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4102,
                                                  "end": 4112
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3820,
                                              "end": 4138
                                            }
                                          },
                                          "loc": {
                                            "start": 3806,
                                            "end": 4138
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3521,
                                        "end": 4160
                                      }
                                    },
                                    "loc": {
                                      "start": 3511,
                                      "end": 4160
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3404,
                                  "end": 4178
                                }
                              },
                              "loc": {
                                "start": 3391,
                                "end": 4178
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 4195,
                                  "end": 4207
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
                                        "start": 4230,
                                        "end": 4232
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4230,
                                      "end": 4232
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4253,
                                        "end": 4263
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4253,
                                      "end": 4263
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4284,
                                        "end": 4296
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
                                              "start": 4323,
                                              "end": 4325
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4323,
                                            "end": 4325
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4350,
                                              "end": 4358
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4350,
                                            "end": 4358
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4383,
                                              "end": 4394
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4383,
                                            "end": 4394
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4419,
                                              "end": 4423
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4419,
                                            "end": 4423
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4297,
                                        "end": 4445
                                      }
                                    },
                                    "loc": {
                                      "start": 4284,
                                      "end": 4445
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 4466,
                                        "end": 4475
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
                                              "start": 4502,
                                              "end": 4504
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4502,
                                            "end": 4504
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 4529,
                                              "end": 4534
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4529,
                                            "end": 4534
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 4559,
                                              "end": 4563
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4559,
                                            "end": 4563
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 4588,
                                              "end": 4595
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4588,
                                            "end": 4595
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 4620,
                                              "end": 4632
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
                                                    "start": 4663,
                                                    "end": 4665
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4663,
                                                  "end": 4665
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 4694,
                                                    "end": 4702
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4694,
                                                  "end": 4702
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 4731,
                                                    "end": 4742
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4731,
                                                  "end": 4742
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 4771,
                                                    "end": 4775
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4771,
                                                  "end": 4775
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 4633,
                                              "end": 4801
                                            }
                                          },
                                          "loc": {
                                            "start": 4620,
                                            "end": 4801
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4476,
                                        "end": 4823
                                      }
                                    },
                                    "loc": {
                                      "start": 4466,
                                      "end": 4823
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4208,
                                  "end": 4841
                                }
                              },
                              "loc": {
                                "start": 4195,
                                "end": 4841
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 4858,
                                  "end": 4866
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
                                        "start": 4892,
                                        "end": 4907
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 4889,
                                      "end": 4907
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4867,
                                  "end": 4925
                                }
                              },
                              "loc": {
                                "start": 4858,
                                "end": 4925
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4942,
                                  "end": 4944
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4942,
                                "end": 4944
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 4961,
                                  "end": 4965
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4961,
                                "end": 4965
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 4982,
                                  "end": 4993
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4982,
                                "end": 4993
                              }
                            }
                          ],
                          "loc": {
                            "start": 426,
                            "end": 5007
                          }
                        },
                        "loc": {
                          "start": 421,
                          "end": 5007
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopCondition",
                          "loc": {
                            "start": 5020,
                            "end": 5033
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5020,
                          "end": 5033
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopTime",
                          "loc": {
                            "start": 5046,
                            "end": 5054
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5046,
                          "end": 5054
                        }
                      }
                    ],
                    "loc": {
                      "start": 407,
                      "end": 5064
                    }
                  },
                  "loc": {
                    "start": 391,
                    "end": 5064
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apisCount",
                    "loc": {
                      "start": 5073,
                      "end": 5082
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5073,
                    "end": 5082
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarkLists",
                    "loc": {
                      "start": 5091,
                      "end": 5104
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
                            "start": 5119,
                            "end": 5121
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5119,
                          "end": 5121
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 5134,
                            "end": 5144
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5134,
                          "end": 5144
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 5157,
                            "end": 5167
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5157,
                          "end": 5167
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 5180,
                            "end": 5185
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5180,
                          "end": 5185
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bookmarksCount",
                          "loc": {
                            "start": 5198,
                            "end": 5212
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5198,
                          "end": 5212
                        }
                      }
                    ],
                    "loc": {
                      "start": 5105,
                      "end": 5222
                    }
                  },
                  "loc": {
                    "start": 5091,
                    "end": 5222
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusModes",
                    "loc": {
                      "start": 5231,
                      "end": 5241
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
                            "start": 5256,
                            "end": 5263
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
                                  "start": 5282,
                                  "end": 5284
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5282,
                                "end": 5284
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 5301,
                                  "end": 5311
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5301,
                                "end": 5311
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 5328,
                                  "end": 5331
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
                                        "start": 5354,
                                        "end": 5356
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5354,
                                      "end": 5356
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 5377,
                                        "end": 5387
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5377,
                                      "end": 5387
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 5408,
                                        "end": 5411
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5408,
                                      "end": 5411
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 5432,
                                        "end": 5441
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5432,
                                      "end": 5441
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 5462,
                                        "end": 5474
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
                                              "start": 5501,
                                              "end": 5503
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5501,
                                            "end": 5503
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 5528,
                                              "end": 5536
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5528,
                                            "end": 5536
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5561,
                                              "end": 5572
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5561,
                                            "end": 5572
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5475,
                                        "end": 5594
                                      }
                                    },
                                    "loc": {
                                      "start": 5462,
                                      "end": 5594
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 5615,
                                        "end": 5618
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
                                              "start": 5645,
                                              "end": 5650
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5645,
                                            "end": 5650
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 5675,
                                              "end": 5687
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5675,
                                            "end": 5687
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5619,
                                        "end": 5709
                                      }
                                    },
                                    "loc": {
                                      "start": 5615,
                                      "end": 5709
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5332,
                                  "end": 5727
                                }
                              },
                              "loc": {
                                "start": 5328,
                                "end": 5727
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 5744,
                                  "end": 5753
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
                                        "start": 5776,
                                        "end": 5782
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
                                              "start": 5809,
                                              "end": 5811
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5809,
                                            "end": 5811
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 5836,
                                              "end": 5841
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5836,
                                            "end": 5841
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 5866,
                                              "end": 5871
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5866,
                                            "end": 5871
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5783,
                                        "end": 5893
                                      }
                                    },
                                    "loc": {
                                      "start": 5776,
                                      "end": 5893
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderList",
                                      "loc": {
                                        "start": 5914,
                                        "end": 5926
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
                                              "start": 5953,
                                              "end": 5955
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5953,
                                            "end": 5955
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 5980,
                                              "end": 5990
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5980,
                                            "end": 5990
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 6015,
                                              "end": 6025
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6015,
                                            "end": 6025
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminders",
                                            "loc": {
                                              "start": 6050,
                                              "end": 6059
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
                                                    "start": 6090,
                                                    "end": 6092
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6090,
                                                  "end": 6092
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 6121,
                                                    "end": 6131
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6121,
                                                  "end": 6131
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 6160,
                                                    "end": 6170
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6160,
                                                  "end": 6170
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 6199,
                                                    "end": 6203
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6199,
                                                  "end": 6203
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 6232,
                                                    "end": 6243
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6232,
                                                  "end": 6243
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 6272,
                                                    "end": 6279
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6272,
                                                  "end": 6279
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 6308,
                                                    "end": 6313
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6308,
                                                  "end": 6313
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 6342,
                                                    "end": 6352
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6342,
                                                  "end": 6352
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminderItems",
                                                  "loc": {
                                                    "start": 6381,
                                                    "end": 6394
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
                                                          "start": 6429,
                                                          "end": 6431
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6429,
                                                        "end": 6431
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 6464,
                                                          "end": 6474
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6464,
                                                        "end": 6474
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 6507,
                                                          "end": 6517
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6507,
                                                        "end": 6517
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 6550,
                                                          "end": 6554
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6550,
                                                        "end": 6554
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 6587,
                                                          "end": 6598
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6587,
                                                        "end": 6598
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 6631,
                                                          "end": 6638
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6631,
                                                        "end": 6638
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 6671,
                                                          "end": 6676
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6671,
                                                        "end": 6676
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 6709,
                                                          "end": 6719
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6709,
                                                        "end": 6719
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 6395,
                                                    "end": 6749
                                                  }
                                                },
                                                "loc": {
                                                  "start": 6381,
                                                  "end": 6749
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6060,
                                              "end": 6775
                                            }
                                          },
                                          "loc": {
                                            "start": 6050,
                                            "end": 6775
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5927,
                                        "end": 6797
                                      }
                                    },
                                    "loc": {
                                      "start": 5914,
                                      "end": 6797
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resourceList",
                                      "loc": {
                                        "start": 6818,
                                        "end": 6830
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
                                              "start": 6857,
                                              "end": 6859
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6857,
                                            "end": 6859
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 6884,
                                              "end": 6894
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6884,
                                            "end": 6894
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 6919,
                                              "end": 6931
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
                                                    "start": 6962,
                                                    "end": 6964
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6962,
                                                  "end": 6964
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 6993,
                                                    "end": 7001
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6993,
                                                  "end": 7001
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 7030,
                                                    "end": 7041
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7030,
                                                  "end": 7041
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 7070,
                                                    "end": 7074
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7070,
                                                  "end": 7074
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6932,
                                              "end": 7100
                                            }
                                          },
                                          "loc": {
                                            "start": 6919,
                                            "end": 7100
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resources",
                                            "loc": {
                                              "start": 7125,
                                              "end": 7134
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
                                                    "start": 7165,
                                                    "end": 7167
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7165,
                                                  "end": 7167
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 7196,
                                                    "end": 7201
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7196,
                                                  "end": 7201
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "link",
                                                  "loc": {
                                                    "start": 7230,
                                                    "end": 7234
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7230,
                                                  "end": 7234
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "usedFor",
                                                  "loc": {
                                                    "start": 7263,
                                                    "end": 7270
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7263,
                                                  "end": 7270
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 7299,
                                                    "end": 7311
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
                                                          "start": 7346,
                                                          "end": 7348
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7346,
                                                        "end": 7348
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 7381,
                                                          "end": 7389
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7381,
                                                        "end": 7389
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 7422,
                                                          "end": 7433
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7422,
                                                        "end": 7433
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 7466,
                                                          "end": 7470
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7466,
                                                        "end": 7470
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 7312,
                                                    "end": 7500
                                                  }
                                                },
                                                "loc": {
                                                  "start": 7299,
                                                  "end": 7500
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 7135,
                                              "end": 7526
                                            }
                                          },
                                          "loc": {
                                            "start": 7125,
                                            "end": 7526
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6831,
                                        "end": 7548
                                      }
                                    },
                                    "loc": {
                                      "start": 6818,
                                      "end": 7548
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 7569,
                                        "end": 7577
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
                                              "start": 7607,
                                              "end": 7622
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 7604,
                                            "end": 7622
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 7578,
                                        "end": 7644
                                      }
                                    },
                                    "loc": {
                                      "start": 7569,
                                      "end": 7644
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 7665,
                                        "end": 7667
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7665,
                                      "end": 7667
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 7688,
                                        "end": 7692
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7688,
                                      "end": 7692
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 7713,
                                        "end": 7724
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7713,
                                      "end": 7724
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5754,
                                  "end": 7742
                                }
                              },
                              "loc": {
                                "start": 5744,
                                "end": 7742
                              }
                            }
                          ],
                          "loc": {
                            "start": 5264,
                            "end": 7756
                          }
                        },
                        "loc": {
                          "start": 5256,
                          "end": 7756
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 7769,
                            "end": 7775
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
                                  "start": 7794,
                                  "end": 7796
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7794,
                                "end": 7796
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 7813,
                                  "end": 7818
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7813,
                                "end": 7818
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 7835,
                                  "end": 7840
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7835,
                                "end": 7840
                              }
                            }
                          ],
                          "loc": {
                            "start": 7776,
                            "end": 7854
                          }
                        },
                        "loc": {
                          "start": 7769,
                          "end": 7854
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 7867,
                            "end": 7879
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
                                  "start": 7898,
                                  "end": 7900
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7898,
                                "end": 7900
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 7917,
                                  "end": 7927
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7917,
                                "end": 7927
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 7944,
                                  "end": 7954
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7944,
                                "end": 7954
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 7971,
                                  "end": 7980
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
                                        "start": 8003,
                                        "end": 8005
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8003,
                                      "end": 8005
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 8026,
                                        "end": 8036
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8026,
                                      "end": 8036
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 8057,
                                        "end": 8067
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8057,
                                      "end": 8067
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 8088,
                                        "end": 8092
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8088,
                                      "end": 8092
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 8113,
                                        "end": 8124
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8113,
                                      "end": 8124
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 8145,
                                        "end": 8152
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8145,
                                      "end": 8152
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 8173,
                                        "end": 8178
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8173,
                                      "end": 8178
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 8199,
                                        "end": 8209
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8199,
                                      "end": 8209
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 8230,
                                        "end": 8243
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
                                              "start": 8270,
                                              "end": 8272
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8270,
                                            "end": 8272
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 8297,
                                              "end": 8307
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8297,
                                            "end": 8307
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 8332,
                                              "end": 8342
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8332,
                                            "end": 8342
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 8367,
                                              "end": 8371
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8367,
                                            "end": 8371
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 8396,
                                              "end": 8407
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8396,
                                            "end": 8407
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 8432,
                                              "end": 8439
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8432,
                                            "end": 8439
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 8464,
                                              "end": 8469
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8464,
                                            "end": 8469
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 8494,
                                              "end": 8504
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8494,
                                            "end": 8504
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 8244,
                                        "end": 8526
                                      }
                                    },
                                    "loc": {
                                      "start": 8230,
                                      "end": 8526
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 7981,
                                  "end": 8544
                                }
                              },
                              "loc": {
                                "start": 7971,
                                "end": 8544
                              }
                            }
                          ],
                          "loc": {
                            "start": 7880,
                            "end": 8558
                          }
                        },
                        "loc": {
                          "start": 7867,
                          "end": 8558
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 8571,
                            "end": 8583
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
                                  "start": 8602,
                                  "end": 8604
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8602,
                                "end": 8604
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 8621,
                                  "end": 8631
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8621,
                                "end": 8631
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 8648,
                                  "end": 8660
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
                                        "start": 8683,
                                        "end": 8685
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8683,
                                      "end": 8685
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 8706,
                                        "end": 8714
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8706,
                                      "end": 8714
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 8735,
                                        "end": 8746
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8735,
                                      "end": 8746
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 8767,
                                        "end": 8771
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8767,
                                      "end": 8771
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8661,
                                  "end": 8789
                                }
                              },
                              "loc": {
                                "start": 8648,
                                "end": 8789
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 8806,
                                  "end": 8815
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
                                        "start": 8838,
                                        "end": 8840
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8838,
                                      "end": 8840
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 8861,
                                        "end": 8866
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8861,
                                      "end": 8866
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 8887,
                                        "end": 8891
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8887,
                                      "end": 8891
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 8912,
                                        "end": 8919
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8912,
                                      "end": 8919
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 8940,
                                        "end": 8952
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
                                              "start": 8979,
                                              "end": 8981
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8979,
                                            "end": 8981
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 9006,
                                              "end": 9014
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9006,
                                            "end": 9014
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 9039,
                                              "end": 9050
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9039,
                                            "end": 9050
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 9075,
                                              "end": 9079
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9075,
                                            "end": 9079
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 8953,
                                        "end": 9101
                                      }
                                    },
                                    "loc": {
                                      "start": 8940,
                                      "end": 9101
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8816,
                                  "end": 9119
                                }
                              },
                              "loc": {
                                "start": 8806,
                                "end": 9119
                              }
                            }
                          ],
                          "loc": {
                            "start": 8584,
                            "end": 9133
                          }
                        },
                        "loc": {
                          "start": 8571,
                          "end": 9133
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 9146,
                            "end": 9154
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
                                  "start": 9176,
                                  "end": 9191
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 9173,
                                "end": 9191
                              }
                            }
                          ],
                          "loc": {
                            "start": 9155,
                            "end": 9205
                          }
                        },
                        "loc": {
                          "start": 9146,
                          "end": 9205
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 9218,
                            "end": 9220
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9218,
                          "end": 9220
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 9233,
                            "end": 9237
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9233,
                          "end": 9237
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 9250,
                            "end": 9261
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9250,
                          "end": 9261
                        }
                      }
                    ],
                    "loc": {
                      "start": 5242,
                      "end": 9271
                    }
                  },
                  "loc": {
                    "start": 5231,
                    "end": 9271
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 9280,
                      "end": 9286
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9280,
                    "end": 9286
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 9295,
                      "end": 9305
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9295,
                    "end": 9305
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 9314,
                      "end": 9316
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9314,
                    "end": 9316
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "languages",
                    "loc": {
                      "start": 9325,
                      "end": 9334
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9325,
                    "end": 9334
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 9343,
                      "end": 9359
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9343,
                    "end": 9359
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 9368,
                      "end": 9372
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9368,
                    "end": 9372
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 9381,
                      "end": 9391
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9381,
                    "end": 9391
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 9400,
                      "end": 9413
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9400,
                    "end": 9413
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 9422,
                      "end": 9441
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9422,
                    "end": 9441
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 9450,
                      "end": 9463
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9450,
                    "end": 9463
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractsCount",
                    "loc": {
                      "start": 9472,
                      "end": 9491
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9472,
                    "end": 9491
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 9500,
                      "end": 9514
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9500,
                    "end": 9514
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 9523,
                      "end": 9528
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9523,
                    "end": 9528
                  }
                }
              ],
              "loc": {
                "start": 381,
                "end": 9534
              }
            },
            "loc": {
              "start": 375,
              "end": 9534
            }
          }
        ],
        "loc": {
          "start": 341,
          "end": 9538
        }
      },
      "loc": {
        "start": 319,
        "end": 9538
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
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "logOut",
      "loc": {
        "start": 286,
        "end": 292
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
              "start": 294,
              "end": 299
            }
          },
          "loc": {
            "start": 293,
            "end": 299
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "LogOutInput",
              "loc": {
                "start": 301,
                "end": 312
              }
            },
            "loc": {
              "start": 301,
              "end": 312
            }
          },
          "loc": {
            "start": 301,
            "end": 313
          }
        },
        "directives": [],
        "loc": {
          "start": 293,
          "end": 313
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
            "value": "logOut",
            "loc": {
              "start": 319,
              "end": 325
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 326,
                  "end": 331
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 334,
                    "end": 339
                  }
                },
                "loc": {
                  "start": 333,
                  "end": 339
                }
              },
              "loc": {
                "start": 326,
                "end": 339
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
                  "value": "isLoggedIn",
                  "loc": {
                    "start": 347,
                    "end": 357
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 347,
                  "end": 357
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "timeZone",
                  "loc": {
                    "start": 362,
                    "end": 370
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 362,
                  "end": 370
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "users",
                  "loc": {
                    "start": 375,
                    "end": 380
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
                          "start": 391,
                          "end": 406
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
                                "start": 421,
                                "end": 425
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
                                      "start": 444,
                                      "end": 451
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
                                            "start": 474,
                                            "end": 476
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 474,
                                          "end": 476
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "filterType",
                                          "loc": {
                                            "start": 497,
                                            "end": 507
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 497,
                                          "end": 507
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 528,
                                            "end": 531
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
                                                  "start": 558,
                                                  "end": 560
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 558,
                                                "end": 560
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 585,
                                                  "end": 595
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 585,
                                                "end": 595
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "tag",
                                                "loc": {
                                                  "start": 620,
                                                  "end": 623
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 620,
                                                "end": 623
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "bookmarks",
                                                "loc": {
                                                  "start": 648,
                                                  "end": 657
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 648,
                                                "end": 657
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 682,
                                                  "end": 694
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
                                                        "start": 725,
                                                        "end": 727
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 725,
                                                      "end": 727
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 756,
                                                        "end": 764
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 756,
                                                      "end": 764
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 793,
                                                        "end": 804
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 793,
                                                      "end": 804
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 695,
                                                  "end": 830
                                                }
                                              },
                                              "loc": {
                                                "start": 682,
                                                "end": 830
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "you",
                                                "loc": {
                                                  "start": 855,
                                                  "end": 858
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
                                                        "start": 889,
                                                        "end": 894
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 889,
                                                      "end": 894
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isBookmarked",
                                                      "loc": {
                                                        "start": 923,
                                                        "end": 935
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 923,
                                                      "end": 935
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 859,
                                                  "end": 961
                                                }
                                              },
                                              "loc": {
                                                "start": 855,
                                                "end": 961
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 532,
                                            "end": 983
                                          }
                                        },
                                        "loc": {
                                          "start": 528,
                                          "end": 983
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "focusMode",
                                          "loc": {
                                            "start": 1004,
                                            "end": 1013
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
                                                  "start": 1040,
                                                  "end": 1046
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
                                                        "start": 1077,
                                                        "end": 1079
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1077,
                                                      "end": 1079
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "color",
                                                      "loc": {
                                                        "start": 1108,
                                                        "end": 1113
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1108,
                                                      "end": 1113
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "label",
                                                      "loc": {
                                                        "start": 1142,
                                                        "end": 1147
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1142,
                                                      "end": 1147
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1047,
                                                  "end": 1173
                                                }
                                              },
                                              "loc": {
                                                "start": 1040,
                                                "end": 1173
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderList",
                                                "loc": {
                                                  "start": 1198,
                                                  "end": 1210
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
                                                        "start": 1241,
                                                        "end": 1243
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1241,
                                                      "end": 1243
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 1272,
                                                        "end": 1282
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1272,
                                                      "end": 1282
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 1311,
                                                        "end": 1321
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1311,
                                                      "end": 1321
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "reminders",
                                                      "loc": {
                                                        "start": 1350,
                                                        "end": 1359
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
                                                              "start": 1394,
                                                              "end": 1396
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1394,
                                                            "end": 1396
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "created_at",
                                                            "loc": {
                                                              "start": 1429,
                                                              "end": 1439
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1429,
                                                            "end": 1439
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "updated_at",
                                                            "loc": {
                                                              "start": 1472,
                                                              "end": 1482
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1472,
                                                            "end": 1482
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 1515,
                                                              "end": 1519
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1515,
                                                            "end": 1519
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 1552,
                                                              "end": 1563
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1552,
                                                            "end": 1563
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "dueDate",
                                                            "loc": {
                                                              "start": 1596,
                                                              "end": 1603
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1596,
                                                            "end": 1603
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 1636,
                                                              "end": 1641
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1636,
                                                            "end": 1641
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "isComplete",
                                                            "loc": {
                                                              "start": 1674,
                                                              "end": 1684
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1674,
                                                            "end": 1684
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "reminderItems",
                                                            "loc": {
                                                              "start": 1717,
                                                              "end": 1730
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
                                                                    "start": 1769,
                                                                    "end": 1771
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1769,
                                                                  "end": 1771
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "created_at",
                                                                  "loc": {
                                                                    "start": 1808,
                                                                    "end": 1818
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1808,
                                                                  "end": 1818
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "updated_at",
                                                                  "loc": {
                                                                    "start": 1855,
                                                                    "end": 1865
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1855,
                                                                  "end": 1865
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "name",
                                                                  "loc": {
                                                                    "start": 1902,
                                                                    "end": 1906
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1902,
                                                                  "end": 1906
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "description",
                                                                  "loc": {
                                                                    "start": 1943,
                                                                    "end": 1954
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1943,
                                                                  "end": 1954
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "dueDate",
                                                                  "loc": {
                                                                    "start": 1991,
                                                                    "end": 1998
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1991,
                                                                  "end": 1998
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "index",
                                                                  "loc": {
                                                                    "start": 2035,
                                                                    "end": 2040
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2035,
                                                                  "end": 2040
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "isComplete",
                                                                  "loc": {
                                                                    "start": 2077,
                                                                    "end": 2087
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2077,
                                                                  "end": 2087
                                                                }
                                                              }
                                                            ],
                                                            "loc": {
                                                              "start": 1731,
                                                              "end": 2121
                                                            }
                                                          },
                                                          "loc": {
                                                            "start": 1717,
                                                            "end": 2121
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 1360,
                                                        "end": 2151
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 1350,
                                                      "end": 2151
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1211,
                                                  "end": 2177
                                                }
                                              },
                                              "loc": {
                                                "start": 1198,
                                                "end": 2177
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "resourceList",
                                                "loc": {
                                                  "start": 2202,
                                                  "end": 2214
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
                                                        "start": 2245,
                                                        "end": 2247
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2245,
                                                      "end": 2247
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 2276,
                                                        "end": 2286
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2276,
                                                      "end": 2286
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "translations",
                                                      "loc": {
                                                        "start": 2315,
                                                        "end": 2327
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
                                                              "start": 2362,
                                                              "end": 2364
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2362,
                                                            "end": 2364
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "language",
                                                            "loc": {
                                                              "start": 2397,
                                                              "end": 2405
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2397,
                                                            "end": 2405
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 2438,
                                                              "end": 2449
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2438,
                                                            "end": 2449
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 2482,
                                                              "end": 2486
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2482,
                                                            "end": 2486
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 2328,
                                                        "end": 2516
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 2315,
                                                      "end": 2516
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "resources",
                                                      "loc": {
                                                        "start": 2545,
                                                        "end": 2554
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
                                                              "start": 2589,
                                                              "end": 2591
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2589,
                                                            "end": 2591
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 2624,
                                                              "end": 2629
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2624,
                                                            "end": 2629
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "link",
                                                            "loc": {
                                                              "start": 2662,
                                                              "end": 2666
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2662,
                                                            "end": 2666
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "usedFor",
                                                            "loc": {
                                                              "start": 2699,
                                                              "end": 2706
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2699,
                                                            "end": 2706
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "translations",
                                                            "loc": {
                                                              "start": 2739,
                                                              "end": 2751
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
                                                                    "start": 2790,
                                                                    "end": 2792
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2790,
                                                                  "end": 2792
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "language",
                                                                  "loc": {
                                                                    "start": 2829,
                                                                    "end": 2837
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2829,
                                                                  "end": 2837
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "description",
                                                                  "loc": {
                                                                    "start": 2874,
                                                                    "end": 2885
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2874,
                                                                  "end": 2885
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "name",
                                                                  "loc": {
                                                                    "start": 2922,
                                                                    "end": 2926
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2922,
                                                                  "end": 2926
                                                                }
                                                              }
                                                            ],
                                                            "loc": {
                                                              "start": 2752,
                                                              "end": 2960
                                                            }
                                                          },
                                                          "loc": {
                                                            "start": 2739,
                                                            "end": 2960
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 2555,
                                                        "end": 2990
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 2545,
                                                      "end": 2990
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 2215,
                                                  "end": 3016
                                                }
                                              },
                                              "loc": {
                                                "start": 2202,
                                                "end": 3016
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "schedule",
                                                "loc": {
                                                  "start": 3041,
                                                  "end": 3049
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
                                                        "start": 3083,
                                                        "end": 3098
                                                      }
                                                    },
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3080,
                                                      "end": 3098
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 3050,
                                                  "end": 3124
                                                }
                                              },
                                              "loc": {
                                                "start": 3041,
                                                "end": 3124
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 3149,
                                                  "end": 3151
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3149,
                                                "end": 3151
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 3176,
                                                  "end": 3180
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3176,
                                                "end": 3180
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3205,
                                                  "end": 3216
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3205,
                                                "end": 3216
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1014,
                                            "end": 3238
                                          }
                                        },
                                        "loc": {
                                          "start": 1004,
                                          "end": 3238
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 452,
                                      "end": 3256
                                    }
                                  },
                                  "loc": {
                                    "start": 444,
                                    "end": 3256
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "labels",
                                    "loc": {
                                      "start": 3273,
                                      "end": 3279
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
                                            "start": 3302,
                                            "end": 3304
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3302,
                                          "end": 3304
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "color",
                                          "loc": {
                                            "start": 3325,
                                            "end": 3330
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3325,
                                          "end": 3330
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "label",
                                          "loc": {
                                            "start": 3351,
                                            "end": 3356
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3351,
                                          "end": 3356
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3280,
                                      "end": 3374
                                    }
                                  },
                                  "loc": {
                                    "start": 3273,
                                    "end": 3374
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminderList",
                                    "loc": {
                                      "start": 3391,
                                      "end": 3403
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
                                            "start": 3426,
                                            "end": 3428
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3426,
                                          "end": 3428
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 3449,
                                            "end": 3459
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3449,
                                          "end": 3459
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 3480,
                                            "end": 3490
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3480,
                                          "end": 3490
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminders",
                                          "loc": {
                                            "start": 3511,
                                            "end": 3520
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
                                                  "start": 3547,
                                                  "end": 3549
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3547,
                                                "end": 3549
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 3574,
                                                  "end": 3584
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3574,
                                                "end": 3584
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 3609,
                                                  "end": 3619
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3609,
                                                "end": 3619
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 3644,
                                                  "end": 3648
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3644,
                                                "end": 3648
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3673,
                                                  "end": 3684
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3673,
                                                "end": 3684
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 3709,
                                                  "end": 3716
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3709,
                                                "end": 3716
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 3741,
                                                  "end": 3746
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3741,
                                                "end": 3746
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 3771,
                                                  "end": 3781
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3771,
                                                "end": 3781
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderItems",
                                                "loc": {
                                                  "start": 3806,
                                                  "end": 3819
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
                                                        "start": 3850,
                                                        "end": 3852
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3850,
                                                      "end": 3852
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 3881,
                                                        "end": 3891
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3881,
                                                      "end": 3891
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 3920,
                                                        "end": 3930
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3920,
                                                      "end": 3930
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 3959,
                                                        "end": 3963
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3959,
                                                      "end": 3963
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 3992,
                                                        "end": 4003
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3992,
                                                      "end": 4003
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 4032,
                                                        "end": 4039
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4032,
                                                      "end": 4039
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 4068,
                                                        "end": 4073
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4068,
                                                      "end": 4073
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 4102,
                                                        "end": 4112
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4102,
                                                      "end": 4112
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 3820,
                                                  "end": 4138
                                                }
                                              },
                                              "loc": {
                                                "start": 3806,
                                                "end": 4138
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3521,
                                            "end": 4160
                                          }
                                        },
                                        "loc": {
                                          "start": 3511,
                                          "end": 4160
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3404,
                                      "end": 4178
                                    }
                                  },
                                  "loc": {
                                    "start": 3391,
                                    "end": 4178
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resourceList",
                                    "loc": {
                                      "start": 4195,
                                      "end": 4207
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
                                            "start": 4230,
                                            "end": 4232
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4230,
                                          "end": 4232
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 4253,
                                            "end": 4263
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4253,
                                          "end": 4263
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 4284,
                                            "end": 4296
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
                                                  "start": 4323,
                                                  "end": 4325
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4323,
                                                "end": 4325
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 4350,
                                                  "end": 4358
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4350,
                                                "end": 4358
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 4383,
                                                  "end": 4394
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4383,
                                                "end": 4394
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 4419,
                                                  "end": 4423
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4419,
                                                "end": 4423
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4297,
                                            "end": 4445
                                          }
                                        },
                                        "loc": {
                                          "start": 4284,
                                          "end": 4445
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resources",
                                          "loc": {
                                            "start": 4466,
                                            "end": 4475
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
                                                  "start": 4502,
                                                  "end": 4504
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4502,
                                                "end": 4504
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 4529,
                                                  "end": 4534
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4529,
                                                "end": 4534
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "link",
                                                "loc": {
                                                  "start": 4559,
                                                  "end": 4563
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4559,
                                                "end": 4563
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "usedFor",
                                                "loc": {
                                                  "start": 4588,
                                                  "end": 4595
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4588,
                                                "end": 4595
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 4620,
                                                  "end": 4632
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
                                                        "start": 4663,
                                                        "end": 4665
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4663,
                                                      "end": 4665
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 4694,
                                                        "end": 4702
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4694,
                                                      "end": 4702
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 4731,
                                                        "end": 4742
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4731,
                                                      "end": 4742
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 4771,
                                                        "end": 4775
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4771,
                                                      "end": 4775
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 4633,
                                                  "end": 4801
                                                }
                                              },
                                              "loc": {
                                                "start": 4620,
                                                "end": 4801
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4476,
                                            "end": 4823
                                          }
                                        },
                                        "loc": {
                                          "start": 4466,
                                          "end": 4823
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4208,
                                      "end": 4841
                                    }
                                  },
                                  "loc": {
                                    "start": 4195,
                                    "end": 4841
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "schedule",
                                    "loc": {
                                      "start": 4858,
                                      "end": 4866
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
                                            "start": 4892,
                                            "end": 4907
                                          }
                                        },
                                        "directives": [],
                                        "loc": {
                                          "start": 4889,
                                          "end": 4907
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4867,
                                      "end": 4925
                                    }
                                  },
                                  "loc": {
                                    "start": 4858,
                                    "end": 4925
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 4942,
                                      "end": 4944
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4942,
                                    "end": 4944
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "name",
                                    "loc": {
                                      "start": 4961,
                                      "end": 4965
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4961,
                                    "end": 4965
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "description",
                                    "loc": {
                                      "start": 4982,
                                      "end": 4993
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4982,
                                    "end": 4993
                                  }
                                }
                              ],
                              "loc": {
                                "start": 426,
                                "end": 5007
                              }
                            },
                            "loc": {
                              "start": 421,
                              "end": 5007
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopCondition",
                              "loc": {
                                "start": 5020,
                                "end": 5033
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5020,
                              "end": 5033
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopTime",
                              "loc": {
                                "start": 5046,
                                "end": 5054
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5046,
                              "end": 5054
                            }
                          }
                        ],
                        "loc": {
                          "start": 407,
                          "end": 5064
                        }
                      },
                      "loc": {
                        "start": 391,
                        "end": 5064
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "apisCount",
                        "loc": {
                          "start": 5073,
                          "end": 5082
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5073,
                        "end": 5082
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "bookmarkLists",
                        "loc": {
                          "start": 5091,
                          "end": 5104
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
                                "start": 5119,
                                "end": 5121
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5119,
                              "end": 5121
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "created_at",
                              "loc": {
                                "start": 5134,
                                "end": 5144
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5134,
                              "end": 5144
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "updated_at",
                              "loc": {
                                "start": 5157,
                                "end": 5167
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5157,
                              "end": 5167
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "label",
                              "loc": {
                                "start": 5180,
                                "end": 5185
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5180,
                              "end": 5185
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "bookmarksCount",
                              "loc": {
                                "start": 5198,
                                "end": 5212
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5198,
                              "end": 5212
                            }
                          }
                        ],
                        "loc": {
                          "start": 5105,
                          "end": 5222
                        }
                      },
                      "loc": {
                        "start": 5091,
                        "end": 5222
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "focusModes",
                        "loc": {
                          "start": 5231,
                          "end": 5241
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
                                "start": 5256,
                                "end": 5263
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
                                      "start": 5282,
                                      "end": 5284
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5282,
                                    "end": 5284
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "filterType",
                                    "loc": {
                                      "start": 5301,
                                      "end": 5311
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5301,
                                    "end": 5311
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "tag",
                                    "loc": {
                                      "start": 5328,
                                      "end": 5331
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
                                            "start": 5354,
                                            "end": 5356
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5354,
                                          "end": 5356
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 5377,
                                            "end": 5387
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5377,
                                          "end": 5387
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 5408,
                                            "end": 5411
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5408,
                                          "end": 5411
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "bookmarks",
                                          "loc": {
                                            "start": 5432,
                                            "end": 5441
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5432,
                                          "end": 5441
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 5462,
                                            "end": 5474
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
                                                  "start": 5501,
                                                  "end": 5503
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5501,
                                                "end": 5503
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 5528,
                                                  "end": 5536
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5528,
                                                "end": 5536
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 5561,
                                                  "end": 5572
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5561,
                                                "end": 5572
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5475,
                                            "end": 5594
                                          }
                                        },
                                        "loc": {
                                          "start": 5462,
                                          "end": 5594
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "you",
                                          "loc": {
                                            "start": 5615,
                                            "end": 5618
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
                                                  "start": 5645,
                                                  "end": 5650
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5645,
                                                "end": 5650
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isBookmarked",
                                                "loc": {
                                                  "start": 5675,
                                                  "end": 5687
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5675,
                                                "end": 5687
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5619,
                                            "end": 5709
                                          }
                                        },
                                        "loc": {
                                          "start": 5615,
                                          "end": 5709
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5332,
                                      "end": 5727
                                    }
                                  },
                                  "loc": {
                                    "start": 5328,
                                    "end": 5727
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "focusMode",
                                    "loc": {
                                      "start": 5744,
                                      "end": 5753
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
                                            "start": 5776,
                                            "end": 5782
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
                                                  "start": 5809,
                                                  "end": 5811
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5809,
                                                "end": 5811
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "color",
                                                "loc": {
                                                  "start": 5836,
                                                  "end": 5841
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5836,
                                                "end": 5841
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "label",
                                                "loc": {
                                                  "start": 5866,
                                                  "end": 5871
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5866,
                                                "end": 5871
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5783,
                                            "end": 5893
                                          }
                                        },
                                        "loc": {
                                          "start": 5776,
                                          "end": 5893
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderList",
                                          "loc": {
                                            "start": 5914,
                                            "end": 5926
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
                                                  "start": 5953,
                                                  "end": 5955
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5953,
                                                "end": 5955
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 5980,
                                                  "end": 5990
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5980,
                                                "end": 5990
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 6015,
                                                  "end": 6025
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6015,
                                                "end": 6025
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminders",
                                                "loc": {
                                                  "start": 6050,
                                                  "end": 6059
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
                                                        "start": 6090,
                                                        "end": 6092
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6090,
                                                      "end": 6092
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 6121,
                                                        "end": 6131
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6121,
                                                      "end": 6131
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 6160,
                                                        "end": 6170
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6160,
                                                      "end": 6170
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 6199,
                                                        "end": 6203
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6199,
                                                      "end": 6203
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 6232,
                                                        "end": 6243
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6232,
                                                      "end": 6243
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 6272,
                                                        "end": 6279
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6272,
                                                      "end": 6279
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 6308,
                                                        "end": 6313
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6308,
                                                      "end": 6313
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 6342,
                                                        "end": 6352
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6342,
                                                      "end": 6352
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "reminderItems",
                                                      "loc": {
                                                        "start": 6381,
                                                        "end": 6394
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
                                                              "start": 6429,
                                                              "end": 6431
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6429,
                                                            "end": 6431
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "created_at",
                                                            "loc": {
                                                              "start": 6464,
                                                              "end": 6474
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6464,
                                                            "end": 6474
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "updated_at",
                                                            "loc": {
                                                              "start": 6507,
                                                              "end": 6517
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6507,
                                                            "end": 6517
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 6550,
                                                              "end": 6554
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6550,
                                                            "end": 6554
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 6587,
                                                              "end": 6598
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6587,
                                                            "end": 6598
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "dueDate",
                                                            "loc": {
                                                              "start": 6631,
                                                              "end": 6638
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6631,
                                                            "end": 6638
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 6671,
                                                              "end": 6676
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6671,
                                                            "end": 6676
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "isComplete",
                                                            "loc": {
                                                              "start": 6709,
                                                              "end": 6719
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6709,
                                                            "end": 6719
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 6395,
                                                        "end": 6749
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 6381,
                                                      "end": 6749
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 6060,
                                                  "end": 6775
                                                }
                                              },
                                              "loc": {
                                                "start": 6050,
                                                "end": 6775
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5927,
                                            "end": 6797
                                          }
                                        },
                                        "loc": {
                                          "start": 5914,
                                          "end": 6797
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resourceList",
                                          "loc": {
                                            "start": 6818,
                                            "end": 6830
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
                                                  "start": 6857,
                                                  "end": 6859
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6857,
                                                "end": 6859
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 6884,
                                                  "end": 6894
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6884,
                                                "end": 6894
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 6919,
                                                  "end": 6931
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
                                                        "start": 6962,
                                                        "end": 6964
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6962,
                                                      "end": 6964
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 6993,
                                                        "end": 7001
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6993,
                                                      "end": 7001
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 7030,
                                                        "end": 7041
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7030,
                                                      "end": 7041
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 7070,
                                                        "end": 7074
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7070,
                                                      "end": 7074
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 6932,
                                                  "end": 7100
                                                }
                                              },
                                              "loc": {
                                                "start": 6919,
                                                "end": 7100
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "resources",
                                                "loc": {
                                                  "start": 7125,
                                                  "end": 7134
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
                                                        "start": 7165,
                                                        "end": 7167
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7165,
                                                      "end": 7167
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 7196,
                                                        "end": 7201
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7196,
                                                      "end": 7201
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "link",
                                                      "loc": {
                                                        "start": 7230,
                                                        "end": 7234
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7230,
                                                      "end": 7234
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "usedFor",
                                                      "loc": {
                                                        "start": 7263,
                                                        "end": 7270
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7263,
                                                      "end": 7270
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "translations",
                                                      "loc": {
                                                        "start": 7299,
                                                        "end": 7311
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
                                                              "start": 7346,
                                                              "end": 7348
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7346,
                                                            "end": 7348
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "language",
                                                            "loc": {
                                                              "start": 7381,
                                                              "end": 7389
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7381,
                                                            "end": 7389
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 7422,
                                                              "end": 7433
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7422,
                                                            "end": 7433
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 7466,
                                                              "end": 7470
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7466,
                                                            "end": 7470
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 7312,
                                                        "end": 7500
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 7299,
                                                      "end": 7500
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 7135,
                                                  "end": 7526
                                                }
                                              },
                                              "loc": {
                                                "start": 7125,
                                                "end": 7526
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 6831,
                                            "end": 7548
                                          }
                                        },
                                        "loc": {
                                          "start": 6818,
                                          "end": 7548
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "schedule",
                                          "loc": {
                                            "start": 7569,
                                            "end": 7577
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
                                                  "start": 7607,
                                                  "end": 7622
                                                }
                                              },
                                              "directives": [],
                                              "loc": {
                                                "start": 7604,
                                                "end": 7622
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 7578,
                                            "end": 7644
                                          }
                                        },
                                        "loc": {
                                          "start": 7569,
                                          "end": 7644
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 7665,
                                            "end": 7667
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7665,
                                          "end": 7667
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 7688,
                                            "end": 7692
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7688,
                                          "end": 7692
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 7713,
                                            "end": 7724
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7713,
                                          "end": 7724
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5754,
                                      "end": 7742
                                    }
                                  },
                                  "loc": {
                                    "start": 5744,
                                    "end": 7742
                                  }
                                }
                              ],
                              "loc": {
                                "start": 5264,
                                "end": 7756
                              }
                            },
                            "loc": {
                              "start": 5256,
                              "end": 7756
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "labels",
                              "loc": {
                                "start": 7769,
                                "end": 7775
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
                                      "start": 7794,
                                      "end": 7796
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7794,
                                    "end": 7796
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "color",
                                    "loc": {
                                      "start": 7813,
                                      "end": 7818
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7813,
                                    "end": 7818
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "label",
                                    "loc": {
                                      "start": 7835,
                                      "end": 7840
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7835,
                                    "end": 7840
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7776,
                                "end": 7854
                              }
                            },
                            "loc": {
                              "start": 7769,
                              "end": 7854
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "reminderList",
                              "loc": {
                                "start": 7867,
                                "end": 7879
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
                                      "start": 7898,
                                      "end": 7900
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7898,
                                    "end": 7900
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 7917,
                                      "end": 7927
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7917,
                                    "end": 7927
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "updated_at",
                                    "loc": {
                                      "start": 7944,
                                      "end": 7954
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7944,
                                    "end": 7954
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminders",
                                    "loc": {
                                      "start": 7971,
                                      "end": 7980
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
                                            "start": 8003,
                                            "end": 8005
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8003,
                                          "end": 8005
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 8026,
                                            "end": 8036
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8026,
                                          "end": 8036
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 8057,
                                            "end": 8067
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8057,
                                          "end": 8067
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 8088,
                                            "end": 8092
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8088,
                                          "end": 8092
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 8113,
                                            "end": 8124
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8113,
                                          "end": 8124
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "dueDate",
                                          "loc": {
                                            "start": 8145,
                                            "end": 8152
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8145,
                                          "end": 8152
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 8173,
                                            "end": 8178
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8173,
                                          "end": 8178
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isComplete",
                                          "loc": {
                                            "start": 8199,
                                            "end": 8209
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8199,
                                          "end": 8209
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderItems",
                                          "loc": {
                                            "start": 8230,
                                            "end": 8243
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
                                                  "start": 8270,
                                                  "end": 8272
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8270,
                                                "end": 8272
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 8297,
                                                  "end": 8307
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8297,
                                                "end": 8307
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 8332,
                                                  "end": 8342
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8332,
                                                "end": 8342
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 8367,
                                                  "end": 8371
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8367,
                                                "end": 8371
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 8396,
                                                  "end": 8407
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8396,
                                                "end": 8407
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 8432,
                                                  "end": 8439
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8432,
                                                "end": 8439
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 8464,
                                                  "end": 8469
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8464,
                                                "end": 8469
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 8494,
                                                  "end": 8504
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8494,
                                                "end": 8504
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 8244,
                                            "end": 8526
                                          }
                                        },
                                        "loc": {
                                          "start": 8230,
                                          "end": 8526
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 7981,
                                      "end": 8544
                                    }
                                  },
                                  "loc": {
                                    "start": 7971,
                                    "end": 8544
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7880,
                                "end": 8558
                              }
                            },
                            "loc": {
                              "start": 7867,
                              "end": 8558
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "resourceList",
                              "loc": {
                                "start": 8571,
                                "end": 8583
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
                                      "start": 8602,
                                      "end": 8604
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 8602,
                                    "end": 8604
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 8621,
                                      "end": 8631
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 8621,
                                    "end": 8631
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "translations",
                                    "loc": {
                                      "start": 8648,
                                      "end": 8660
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
                                            "start": 8683,
                                            "end": 8685
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8683,
                                          "end": 8685
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "language",
                                          "loc": {
                                            "start": 8706,
                                            "end": 8714
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8706,
                                          "end": 8714
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 8735,
                                            "end": 8746
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8735,
                                          "end": 8746
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 8767,
                                            "end": 8771
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8767,
                                          "end": 8771
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 8661,
                                      "end": 8789
                                    }
                                  },
                                  "loc": {
                                    "start": 8648,
                                    "end": 8789
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resources",
                                    "loc": {
                                      "start": 8806,
                                      "end": 8815
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
                                            "start": 8838,
                                            "end": 8840
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8838,
                                          "end": 8840
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 8861,
                                            "end": 8866
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8861,
                                          "end": 8866
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "link",
                                          "loc": {
                                            "start": 8887,
                                            "end": 8891
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8887,
                                          "end": 8891
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "usedFor",
                                          "loc": {
                                            "start": 8912,
                                            "end": 8919
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8912,
                                          "end": 8919
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 8940,
                                            "end": 8952
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
                                                  "start": 8979,
                                                  "end": 8981
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8979,
                                                "end": 8981
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 9006,
                                                  "end": 9014
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9006,
                                                "end": 9014
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 9039,
                                                  "end": 9050
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9039,
                                                "end": 9050
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 9075,
                                                  "end": 9079
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9075,
                                                "end": 9079
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 8953,
                                            "end": 9101
                                          }
                                        },
                                        "loc": {
                                          "start": 8940,
                                          "end": 9101
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 8816,
                                      "end": 9119
                                    }
                                  },
                                  "loc": {
                                    "start": 8806,
                                    "end": 9119
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8584,
                                "end": 9133
                              }
                            },
                            "loc": {
                              "start": 8571,
                              "end": 9133
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "schedule",
                              "loc": {
                                "start": 9146,
                                "end": 9154
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
                                      "start": 9176,
                                      "end": 9191
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 9173,
                                    "end": 9191
                                  }
                                }
                              ],
                              "loc": {
                                "start": 9155,
                                "end": 9205
                              }
                            },
                            "loc": {
                              "start": 9146,
                              "end": 9205
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 9218,
                                "end": 9220
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9218,
                              "end": 9220
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "name",
                              "loc": {
                                "start": 9233,
                                "end": 9237
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9233,
                              "end": 9237
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "description",
                              "loc": {
                                "start": 9250,
                                "end": 9261
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9250,
                              "end": 9261
                            }
                          }
                        ],
                        "loc": {
                          "start": 5242,
                          "end": 9271
                        }
                      },
                      "loc": {
                        "start": 5231,
                        "end": 9271
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "handle",
                        "loc": {
                          "start": 9280,
                          "end": 9286
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9280,
                        "end": 9286
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasPremium",
                        "loc": {
                          "start": 9295,
                          "end": 9305
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9295,
                        "end": 9305
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "id",
                        "loc": {
                          "start": 9314,
                          "end": 9316
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9314,
                        "end": 9316
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "languages",
                        "loc": {
                          "start": 9325,
                          "end": 9334
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9325,
                        "end": 9334
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "membershipsCount",
                        "loc": {
                          "start": 9343,
                          "end": 9359
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9343,
                        "end": 9359
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "name",
                        "loc": {
                          "start": 9368,
                          "end": 9372
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9368,
                        "end": 9372
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "notesCount",
                        "loc": {
                          "start": 9381,
                          "end": 9391
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9381,
                        "end": 9391
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "projectsCount",
                        "loc": {
                          "start": 9400,
                          "end": 9413
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9400,
                        "end": 9413
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "questionsAskedCount",
                        "loc": {
                          "start": 9422,
                          "end": 9441
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9422,
                        "end": 9441
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "routinesCount",
                        "loc": {
                          "start": 9450,
                          "end": 9463
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9450,
                        "end": 9463
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "smartContractsCount",
                        "loc": {
                          "start": 9472,
                          "end": 9491
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9472,
                        "end": 9491
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "standardsCount",
                        "loc": {
                          "start": 9500,
                          "end": 9514
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9500,
                        "end": 9514
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "theme",
                        "loc": {
                          "start": 9523,
                          "end": 9528
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9523,
                        "end": 9528
                      }
                    }
                  ],
                  "loc": {
                    "start": 381,
                    "end": 9534
                  }
                },
                "loc": {
                  "start": 375,
                  "end": 9534
                }
              }
            ],
            "loc": {
              "start": 341,
              "end": 9538
            }
          },
          "loc": {
            "start": 319,
            "end": 9538
          }
        }
      ],
      "loc": {
        "start": 315,
        "end": 9540
      }
    },
    "loc": {
      "start": 277,
      "end": 9540
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_logOut"
  }
} as const;
